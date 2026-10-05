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
	err := r.db.WithContext(ctx).Where("video_id  AND account_id = ?", video, accountid).Count(&count).Error
	if err != nil {
		return false, err

	}
	return count > 0, err

}

func (r *LikeRepository) ListLikedVideos(ctx context.Context, accountID uint) ([]Video, error) {
	return nil, nil
}
