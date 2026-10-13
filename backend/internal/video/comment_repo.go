package video

import (
	"context"

	"gorm.io/gorm"
)

type CommentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

func (cr *CommentRepository) GetByID(ctx context.Context, id uint) (*Comment, error) {
	var comment Comment
	if err := cr.db.WithContext(ctx).First(&comment, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &comment, nil
}

func (cr *CommentRepository) DeleteComment(ctx context.Context, comment *Comment) error {
	return cr.db.WithContext(ctx).Delete(comment).Error
}
func (cr *CommentRepository) GetAllComments(ctx context.Context, id uint) ([]Comment, error) {
	var comment []Comment
	if err := cr.db.WithContext(ctx).
		Where("video_id = ?", id).
		Order("created_at asc").
		Find(&comment).
		Limit(200).Error; err != nil {
		return nil, err
	}
	return comment, nil

}
