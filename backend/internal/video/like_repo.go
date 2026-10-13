package video

import (
	"context"

	"gorm.io/gorm"
)

type LikeRepository struct {
	db *gorm.DB
}

func NewLiekRepostiory(db *gorm.DB) *LikeRepository {
	return &LikeRepository{db: db}
}
func (r *LikeRepository) isLike(ctx context.Context, video uint, accountid uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&Like{}).Where("video_id = ? AND account_id = ?", video, accountid).Count(&count).Error
	if err != nil {
		return false, err

	}
	return count > 0, err

}

func (r *LikeRepository) ListLikedVideos(ctx context.Context, accountID uint) ([]Video, error) {

	var videos []Video
	if accountID == 0 {
		return videos, nil
	}
	err := r.db.WithContext(ctx).Model(&Video{}).
		Joins("JOIN likes ON likes.video_id = videos.id").
		Where("likes.account_id = ?", accountID).
		Order("likes.created_at desc").
		Limit(200).Find(&videos).Error
	if err != nil {
		return nil, err
	}
	return videos, nil
}

func (lr *LikeRepository) BatchGetLiked(ctx context.Context, videoIDs []uint, accountID uint) (map[uint]bool, error) {
	likeMap := make(map[uint]bool)
	if len(videoIDs) == 0 {
		return likeMap, nil
	}
	if accountID == 0 {
		return likeMap, nil

	}
	var liked []Like
	err := lr.db.WithContext(ctx).Model(&Like{}).Where("video_id IN ? AND account_id = ?", videoIDs, accountID).
		Find(&liked).Error
	if err != nil {
		return nil, err
	}
	for _, like := range liked {
		likeMap[like.VideoID] = true
	}
	return likeMap, nil
}
