package social

import (
	"context"
	"feedsystem_video_go/internal/account"

	"gorm.io/gorm"
)

type SocialRepository struct {
	db *gorm.DB
}

func NewSocialRepository(db *gorm.DB) *SocialRepository {
	return &SocialRepository{db: db}
}

// ok
func (r *SocialRepository) IsFollowed(ctx context.Context, social *Social) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Where("follower_id = ? AND vlogger_id = ?", social.FollowerID, social.VloggerID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
func (r *SocialRepository) Foll(ctx context.Context, social *Social) error {
	return r.db.WithContext(ctx).Create(social).Error
}

func (r *SocialRepository) UnFoll(ctx context.Context, social *Social) error {
	return r.db.WithContext(ctx).Where("follower_id = ? AND vlogger_id = ?", social.FollowerID, social.VloggerID).
		Delete(&Social{}).Error
}
func (r *SocialRepository) GetAllFoll(ctx context.Context, id uint) ([]*account.Account, error) {
	var allfoll []Social
	if err := r.db.WithContext(ctx).Model(&Social{}).Where("vlogger_id = ?", id).Find(&allfoll).Limit(200).Error; err != nil {
		return nil, err
	}
	follwersID := make([]uint, 0, len(allfoll))
	for _, rel := range allfoll {
		follwersID = append(follwersID, rel.FollowerID)
	}
	if len(follwersID) == 0 {
		return []*account.Account{}, nil

	}
	var follwerID []*account.Account
	if err := r.db.WithContext(ctx).Where("id IN ?", follwersID).Model(&account.Account{}).Find(&follwerID).Error; err != nil {
		return nil, err

	}
	return follwerID, nil
}
func (r *SocialRepository) GetAllVloggers(ctx context.Context, FollowerID uint) ([]*account.Account, error) {
	var relation []Social
	if err := r.db.WithContext(ctx).Model(&Social{}).Where("follower_id = ?", FollowerID).Find(&relation).Limit(200).Error; err != nil {
		return nil, err
	}
	vloggerID := make([]uint, 0, len(relation))

	for _, rel := range relation {
		vloggerID = append(vloggerID, rel.VloggerID)
	}
	if len(vloggerID) == 0 {
		return []*account.Account{}, nil
	}
	var vlogger []*account.Account
	if err := r.db.WithContext(ctx).Where("id IN ?", vloggerID).Model(&account.Account{}).Find(&vlogger).Error; err != nil {
		return nil, err
	}
	return vlogger, nil
}

func (r *SocialRepository) CountVloggers(ctx context.Context, followerID uint) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&Social{}).Where("follower_id = ?", followerID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
func (r *SocialRepository) CountFollowers(ctx context.Context, followerID uint) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&Social{}).Where("vlogger_id = ?", followerID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
