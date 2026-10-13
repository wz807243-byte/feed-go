package feed

import (
	"context"
	"encoding/json"
	rediscache "feedsystem_video_go/internal/middleware/redis"
	"feedsystem_video_go/internal/video"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

type FeedService struct {
	repo         *FeedRepository
	likeRepo     *video.LikeRepository
	rediscache   *rediscache.Client
	localcache   *cache.Cache
	cacheTTL     time.Duration
	requestGroup singleflight.Group
}

func NewFeedService(repo *FeedRepository, likeRepo *video.LikeRepository, rediscache *rediscache.Client) *FeedService {
	return &FeedService{repo: repo, likeRepo: likeRepo, rediscache: rediscache, localcache: cache.New(3*time.Second, 5*time.Second), cacheTTL: 24 * time.Hour}
}
func (s *FeedService) ListLatest(ctx context.Context, limit int, latestBefore time.Time, viewerAccountID uint) (ListLatestResponse, error) {
	if s.rediscache == nil {
		return s.listLatestFromDB(ctx, limit, latestBefore, viewerAccountID)
	}
	zsetTail, err := s.rediscache.ZRangeWithScores(ctx, s.rediscache.Key("feed:global_timeline"), 0, 0)
	if err != nil {
		return s.listLatestFromDB(ctx, limit, latestBefore, viewerAccountID)
	}
	isZsetEmpty := len(zsetTail) == 0
	if isZsetEmpty {
		sfKey := s.rediscache.Key("sf:fallback:global_timeline_rebuild")
		v, err, _ := s.requestGroup.Do(sfKey, func() (interface{}, error) {
			dbvideos, err := s.repo.ListLatest(ctx, 1000, time.Time{})
			if err != nil {
				return nil, err
			}
			if len(dbvideos) == 0 {
				return "EMPTY_DB", err

			}
			bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			var zElements []redis.Z
			for _, vid := range dbvideos {
				zElements = append(zElements, redis.Z{
					Score:  float64(vid.CreateTime.UnixMilli()),
					Member: fmt.Sprintf("%d", vid.ID),
				})
			}
			s.rediscache.ZAdd(bgCtx, s.rediscache.Key("feed:global_timeline"), zElements...)
			return "SUCCESS", nil
		})
		if err != nil {
			return ListLatestResponse{}, err
		}
		if v == "EMPTY_DB" {
			return ListLatestResponse{HasMore: false}, nil
		}
		return s.ListLatest(ctx, limit, latestBefore, viewerAccountID)
	}

	watermark := int64(zsetTail[0].Score)
	reqTime := time.Now().UnixMilli()
	if !latestBefore.IsZero() {
		reqTime = latestBefore.UnixMilli()
	}
	var baseVideos []*video.Video
	if reqTime < watermark {
		sfKey := s.rediscache.Key("sf:cold:listLatest:%d:%d", limit, reqTime)
		v, err, _ := s.requestGroup.Do(sfKey, func() (interface{}, error) {
			return s.repo.ListLatest(ctx, limit, latestBefore)
		})
		if err != nil {
			return ListLatestResponse{}, err
		}
		baseVideos = v.([]*video.Video)
	}
	maxScore := "+inf"
	if !latestBefore.IsZero() {
		maxScore = fmt.Sprintf("%d", reqTime-1)

	}
	videoIDsStr, err := s.rediscache.ZRevRangeByScore(ctx, s.rediscache.Key("feed:global_timeline"), maxScore, "-inf", 0, int64(limit))
	if err != nil {
		return ListLatestResponse{}, err
	}
	var videoIDs []uint
	for _, idStr := range videoIDsStr {
		if id, err := strconv.ParseUint(idStr, 10, 64); err == nil {
			videoIDs = append(videoIDs, uint(id))
		}
	}
	if len(videoIDs) > 0 {
		baseVideos, err = s.GetVideoByIDs(ctx, videoIDs)
		if err != nil {
			return ListLatestResponse{}, err
		}
	}
	if len(baseVideos) < limit {
		remainlimit := limit - len(baseVideos)

		var cold time.Time
		if len(baseVideos) > 0 {
			cold = baseVideos[len(baseVideos)-1].CreateTime
		} else {
			cold = latestBefore

			sfyek := s.rediscache.Key("sf:stitch:listLatest:%d:%d", remainlimit, cold.UnixMilli())
			v, err, _ := s.requestGroup.Do(sfyek, func() (interface{}, error) {
				return s.repo.ListLatest(ctx, remainlimit, cold)
			})
			if err == nil {
				coldVideos := v.([]*video.Video)
				baseVideos = append(baseVideos, coldVideos...)
			}
		}

	}
	var nextTime int64
	if len(baseVideos) > 0 {
		// 将本页最后一条视频的时间作为下一次请求的游标
		nextTime = baseVideos[len(baseVideos)-1].CreateTime.UnixMilli()
	}
	hasMore := len(baseVideos) == limit

	feedVideos, err := s.buildFeedVideos(ctx, baseVideos, viewerAccountID)
	if err != nil {
		return ListLatestResponse{}, err
	}

	return ListLatestResponse{
		VideoList: feedVideos,
		NextTime:  nextTime,
		HasMore:   hasMore,
	}, nil
}

func (s *FeedService) listLatestFromDB(ctx context.Context, limit int, latestBefore time.Time, viewerAccountID uint) (ListLatestResponse, error) {
	videos, err := s.repo.ListLatest(ctx, limit, latestBefore)
	if err != nil {
		return ListLatestResponse{}, err
	}
	feedVideo, err := s.buildFeedVideos(ctx, videos, viewerAccountID)
	if err != nil {
		return ListLatestResponse{}, err
	}
	var nextTime int64
	if len(videos) > 0 {
		nextTime = videos[len(videos)-1].CreateTime.UnixMilli()
	}
	return ListLatestResponse{
		VideoList: feedVideo,
		NextTime:  nextTime,
		HasMore:   len(videos) == limit,
	}, nil
}

func (s *FeedService) buildFeedVideos(ctx context.Context, videos []*video.Video, viewerAccountID uint) ([]FeedVideoItem, error) {
	feedVideos := make([]FeedVideoItem, 0, len(videos))
	videoIDs := make([]uint, len(videos))
	for i, v := range videos {
		videoIDs[i] = v.ID
	}
	likedMap, err := s.likeRepo.BatchGetLiked(ctx, videoIDs, viewerAccountID)
	if err != nil {
		return nil, err
	}
	for _, video := range videos {
		feedVideos = append(feedVideos, FeedVideoItem{
			ID:          video.ID,
			Author:      FeedAuthor{ID: video.AuthorID, Username: video.Username},
			Title:       video.Title,
			Description: video.Description,
			PlayURL:     video.PlayURL,
			CoverURL:    video.CoverURL,
			CreateTime:  video.CreateTime.Unix(),
			LikesCount:  video.LikesCount,
			IsLiked:     likedMap[video.ID],
		})
	}
	return feedVideos, nil
}

func (s *FeedService) GetVideoByIDs(ctx context.Context, videoIDs []uint) ([]*video.Video, error) {

	if len(videoIDs) == 0 {
		return []*video.Video{}, nil
	}
	videoMap := make(map[uint]*video.Video)
	var missdL1 []uint
	for _, id := range videoIDs {

		cacheKey := s.rediscache.Key("video:entity:%d", id)
		if s.localcache != nil {
			if v, found := s.localcache.Get(cacheKey); found {
				if data, ok := v.(video.Video); ok {
					videoMap[id] = &data
					continue
				}
			}

		}
		missdL1 = append(missdL1, id)
	}
	if len(missdL1) == 0 {
		return s.buildOrderedResult(videoIDs, videoMap), nil
	}
	var missdL2 []uint
	if len(missdL1) > 0 {
		cacheKey := make([]string, len(missdL1))
		for i, id := range missdL1 {
			cacheKey[i] = s.rediscache.Key("video:entity:%d", id)
		}
		cacheCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		result, err := s.rediscache.MGet(cacheCtx, cacheKey...)
		cancel()

		if err == nil {
			for i, res := range result {
				id := missdL1[i]
				if res != nil {
					if str, ok := res.(string); ok {
						var v video.Video
						if err := json.Unmarshal([]byte(str), &v); err != nil {
							videoMap[id] = &v
							if s.localcache != nil {
								s.localcache.Set(cacheKey[i], v, 5*time.Second)
							}
							continue
						}

					}
				}
				missdL2 = append(missdL2, id)
			}
		} else {
			missdL2 = missdL1
			log.Printf("L2 Redis MGet 失败，全部降级到 MySQL: %v", err)
		}

		if len(missdL2) == 0 {
			return s.buildOrderedResult(videoIDs, videoMap), nil
		}

		var wg sync.WaitGroup
		var mu sync.Mutex
		for _, id := range missdL2 {
			wg.Add(1)
			go func(videoID uint) {
				// 将等待计数减 1
				defer wg.Done()
				sfKey := s.rediscache.Key("sf:entity:%d", videoID)

				v, err, _ := s.requestGroup.Do(sfKey, func() (interface{}, error) {
					videoList, err := s.repo.GetByIDs(ctx, []uint{videoID})
					if err != nil || videoList == nil {
						return nil, err
					}
					safeCopy := *videoList[0]
					mcachekey := s.rediscache.Key("video:entity:%d", safeCopy.ID)
					if b, err := json.Marshal(safeCopy); err == nil {
						go func(k string, b []byte) {
							setCtx, setcancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
							defer setcancel()
							s.rediscache.SetBytes(setCtx, k, b, time.Hour)
						}(mcachekey, b)

					}
					return videoList[0], err
				})

				if err == nil && v != nil {

					safeCopy := *(v.(*video.Video))
					mu.Lock()
					videoMap[id] = &safeCopy
					mu.Unlock()
					s.localcache.Set(
						s.rediscache.Key("video:entity:%d", safeCopy.ID),
						safeCopy,
						5*time.Second,
					)
				}

			}(id)
		}
		wg.Wait()

	}
	return s.buildOrderedResult(videoIDs, videoMap), nil
}
func (s *FeedService) buildOrderedResult(orderedIDs []uint, dataMap map[uint]*video.Video) []*video.Video {
	res := make([]*video.Video, 0, len(orderedIDs))
	for _, id := range orderedIDs {
		if v, exits := dataMap[id]; exits && v != nil {
			res = append(res, v)
		}
	}
	return res

}
func (s *FeedService) ListLikesCount(ctx context.Context, limit int, cursor *LikesCountCursor, accountId uint) (ListLikesCountResponse, error) {
	video, err := s.repo.ListLikesCountWithCursor(ctx, limit, cursor)
	if err != nil {
		return ListLikesCountResponse{}, err
	}
	hasMore := len(video) == limit
	feedvideo, err := s.buildFeedVideos(ctx, video, accountId)
	if err != nil {
		return ListLikesCountResponse{}, err
	}
	resp := ListLikesCountResponse{
		VideoList: feedvideo,
		HasMore:   hasMore,
	}
	if len(video) > 0 {
		last := video[len(video)-1]
		nextLikesCountBefore := last.LikesCount
		nextIDBefore := last.ID
		resp.NextLikesCountBefore = &nextLikesCountBefore
		resp.NextIDBefore = &nextIDBefore
	}
	return resp, nil

}
func (s *FeedService) ListByFollowing(ctx context.Context, limit int, latestTime time.Time, id uint) (ListByFollowingResponse, error) {
	doListByFollowingFromDB := func() (ListByFollowingResponse, error) {
		videos, err := s.repo.ListByFollowing(ctx, limit, id, latestTime)
		if err != nil {
			return ListByFollowingResponse{}, err
		}
		var nextTime int64
		if len(videos) > 0 {
			nextTime = videos[len(videos)-1].CreateTime.Unix()
		} else {
			nextTime = 0
		}
		hasMore := len(videos) == limit

		feedVdieo, err := s.buildFeedVideos(ctx, videos, id)
		if err != nil {
			return ListByFollowingResponse{}, err
		}
		resp := ListByFollowingResponse{
			VideoList: feedVdieo,
			NextTime:  nextTime,
			HasMore:   hasMore,
		}
		return resp, err
	}
	var cacheKey string
	if id != 0 && s.rediscache != nil {
		before := int64(0)
		if !latestTime.IsZero() {
			before = latestTime.Unix()
		}
		cacheKey = s.rediscache.Key("feed:listByFollowing:limit=%d:accountID=%d:before=%d", limit, id, before)
		caCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()
		b, err := s.rediscache.GetBytes(caCtx, cacheKey)
		if err == nil {
			var cached ListByFollowingResponse
			if err := json.Unmarshal(b, &cached); err == nil {
				return cached, nil
			}
		}
		if rediscache.IsMiss(err) {
			lockKey := "lock" + cacheKey
			token, locked, _ := s.rediscache.Lock(caCtx, lockKey, 500*time.Millisecond)
			if locked {
				defer func() {
					_ = s.rediscache.Unlock(context.Background(), lockKey, token)

				}()
				if b, err := s.rediscache.GetBytes(caCtx, cacheKey); err == nil {
					var cached ListByFollowingResponse
					if err := json.Unmarshal(b, &cached); err == nil {
						return cached, nil
					}
				} else {
					resp, err := doListByFollowingFromDB()
					if err != nil {
						return ListByFollowingResponse{}, err
					}
					if b, err := json.Marshal(resp); err == nil {
						_ = s.rediscache.SetBytes(caCtx, cacheKey, b, s.cacheTTL)

					}
					return resp, nil
				}
			} else {
				for i := 0; i < 5; i++ {
					time.Sleep(20 * time.Millisecond)
					if b, err := s.rediscache.GetBytes(caCtx, cacheKey); err == nil {
						var cached ListByFollowingResponse
						if err := json.Unmarshal(b, &cached); err != nil {
							return cached, nil
						}
					}
				}

			}
		}

	}
	resp, err := doListByFollowingFromDB()
	if err != nil {
		return ListByFollowingResponse{}, err
	}
	if cacheKey != "" {
		if b, err := json.Marshal(resp); err == nil {
			cacheCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)

			defer cancel()
			_ = s.rediscache.SetBytes(cacheCtx, cacheKey, b, s.cacheTTL)
		}
	}
	return resp, nil
}

func (s *FeedService) ListByPopularity(ctx context.Context, limit int, reqAsOf int64, offset int, viewerAccountID uint, latestPopularity int64, latestBefore time.Time, latestIDBefore uint) (ListByPopularityResponse, error) {
	if s.rediscache != nil {
		asOf := time.Now().UTC().Truncate(time.Minute)
		if reqAsOf > 0 {
			asOf = time.Unix(reqAsOf, 0).UTC().Truncate(time.Millisecond)
		}
		const win = 60
		keys := make([]string, 0, win)
		for i := 0; i < win; i++ {
			keys = append(keys, s.rediscache.Key("hot:video:1m:%s", asOf.Add(-time.Duration(i)*time.Minute).Format("200601021504")))

		}
		dest := s.rediscache.Key("hot:video:merge:1m:%s", asOf.Format("200601021504")) // 快照key：同一个as_of页内复用
		opCtx, cancel := context.WithTimeout(ctx, 80*time.Millisecond)
		defer cancel()

		exists, _ := s.rediscache.Exists(opCtx, dest)
		if !exists {
			_ = s.rediscache.ZUnionStore(opCtx, dest, keys, "SUM")
			_ = s.rediscache.Expire(opCtx, dest, 2*time.Millisecond)
		}
		start := int64(offset)
		stop := start + int64(limit) - 1
		members, err := s.rediscache.ZRevRange(opCtx, dest, start, stop)
		if err == nil && len(members) == 0 {
			if offset > 0 {
				return ListByPopularityResponse{
					VideoList:  []FeedVideoItem{},
					AsOf:       asOf.Unix(),
					NextOffset: offset,
					HasMore:    false,
				}, nil
			}

		}
		if err == nil && len(members) > 0 {
			ids := make([]uint, 0, len(members))
			for _, id := range members {
				u, err := strconv.ParseUint(id, 10, 64)
				if err == nil && u > 0 {
					ids = append(ids, uint(u))
				}
			}
			videos, err := s.repo.GetByIDs(ctx, ids)
			if err == nil {
				byID := make(map[uint]*video.Video, len(videos))
				for _, v := range videos {
					byID[v.ID] = v
				}
				ordered := make([]*video.Video, 0, len(ids))
				for _, id := range ids {
					if v := byID[id]; v != nil {
						ordered = append(ordered, v)
					}
				}
				items, err := s.buildFeedVideos(ctx, ordered, viewerAccountID)
				if err != nil {
					return ListByPopularityResponse{}, err
				}
				resp := ListByPopularityResponse{
					VideoList:  items,
					AsOf:       asOf.Unix(),
					NextOffset: offset + len(items),
					HasMore:    len(items) == limit,
				}
				if len(ordered) > 0 {
					last := ordered[len(ordered)-1]
					nextPopularity := last.Popularity
					nextBefore := last.CreateTime
					nextID := last.ID
					resp.NextLatestPopularity = &nextPopularity
					resp.NextLatestBefore = &nextBefore
					resp.NextLatestIDBefore = &nextID
				}
				return resp, nil
			}
		}
	}
	videos, err := s.repo.ListByPopularity(ctx, limit, latestPopularity, latestBefore, latestIDBefore)
	if err != nil {
		return ListByPopularityResponse{}, err
	}
	items, err := s.buildFeedVideos(ctx, videos, viewerAccountID)
	if err != nil {
		return ListByPopularityResponse{}, err
	}
	resp := ListByPopularityResponse{
		VideoList:  items,
		AsOf:       0,
		NextOffset: 0,
		HasMore:    len(items) == limit,
	}
	if len(videos) > 0 {
		last := videos[len(videos)-1]
		nextPopularity := last.Popularity
		nextBefore := last.CreateTime
		nextID := last.ID
		resp.NextLatestPopularity = &nextPopularity
		resp.NextLatestBefore = &nextBefore
		resp.NextLatestIDBefore = &nextID
	}

	return resp, nil
}

func (f *FeedService) ListByTag(ctx context.Context, tagName string, limit int, viewerAccountID uint) ([]FeedVideoItem, error) {
	videos, err := f.repo.ListByTag(ctx, tagName, limit)
	if err != nil {
		return nil, err
	}
	return f.buildFeedVideos(ctx, videos, viewerAccountID)
}
