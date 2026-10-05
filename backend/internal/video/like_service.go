package video

import (
	"context"
	"errors"
)

type LikeService struct {
	repo      *LikeRepository
	VideoRepo *VideoRepository
}

func NewLikeService(repo *LikeRepository, VideoRepo *VideoRepository) *LikeService {
	return &LikeService{
		repo:      repo,
		VideoRepo: VideoRepo,
	}
}
func (ls *LikeService) Like(ctx context.Context, like *Like) error {
	if like == nil {
		return errors.New("like=nil ")
	}
	if like.AccountID == 0 || like.VideoID == 0 {
		return errors.New("video_id and account_id are required")

	}
	if ls.VideoRepo != nil {
		ok, err := ls.VideoRepo.IsExist(ctx, like.VideoID)
		if err != nil {
			return err
		}
		if !ok {
			return err
		}
	}
	liked, err := ls.repo.isLike(ctx, like.VideoID, like.AccountID)
	if err != nil {
		return err
	}
	if liked {
		return err
	}
	return nil
}

func (s *LikeService) ListLikedVideos(ctx context.Context, accountID uint) ([]Video, error) {
	return s.repo.ListLikedVideos(ctx, accountID)
}
