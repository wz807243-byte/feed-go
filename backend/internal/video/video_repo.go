package video

import (
	"context"

	"gorm.io/gorm"
)

type VideoRepository struct {
	db *gorm.DB
}

func NewVideoRepositor(db *gorm.DB) *VideoRepository {
	return &VideoRepository{
		db: db,
	}
}
func (vr *VideoRepository) CreateVideo(ctx context.Context, video *Video) error {
	return vr.db.WithContext(ctx).Create(&video).Error
}
func (vr *VideoRepository) ListByAuthorID(ctx context.Context, id int64) ([]Video, error) {
	var videos []Video
	if err := vr.db.WithContext(ctx).
		Where("author_id", id).
		Order("create_time desc").
		Limit(200).
		Find(&videos).Error; err != nil {
		return nil, err
	}
	return videos, nil
}
func (vr *VideoRepository) IsExist(ctx context.Context, id uint) (bool, error) {
	var videoid Video
	if err := vr.db.WithContext(ctx).First(&videoid, id).Error; err != nil {
		return false, err
	}
	return true, nil
}

