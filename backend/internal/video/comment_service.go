package video

import (
	"context"
	"errors"
	rediscache "feedsystem_video_go/internal/middleware/redis"
	"strings"

	"gorm.io/gorm"
)

type CommentService struct {
	repo            *CommentRepository
	VideoRepository *VideoRepository
	cache           *rediscache.Client
}

func NewCommentService(repo *CommentRepository,
	VideoRepository *VideoRepository,
	cache *rediscache.Client) *CommentService {
	return &CommentService{repo: repo, VideoRepository: VideoRepository, cache: cache}
}

func (cs *CommentService) Publish(ctx context.Context, comment *Comment) error {
	if comment == nil {
		return errors.New("comment is nil")
	}
	comment.Username = strings.TrimSpace(comment.Username)
	comment.Content = strings.TrimSpace(comment.Content)
	if comment.AuthorID == 0 || comment.VideoID == 0 {
		return errors.New(" comment.AuthorID == 0  And comment.VideoID == 0")

	}
	if comment.Content == "" {
		return errors.New("content is nil")

	}
	exists, err := cs.VideoRepository.IsExist(ctx, comment.VideoID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New(" Nofound")
	}
	mysqlEnqueued := false
	redisEnqueued := false
	if !mysqlEnqueued {
		if err := cs.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Select("id").First(&Video{}, comment.VideoID).Error; err != nil {
				return errors.ErrUnsupported
			}
			if err := cs.repo.db.Create(comment).Error; err != nil {
				return err
			}
			return tx.Model(&Video{}).Where("id", comment.VideoID).UpdateColumn("popularity", gorm.Expr("popularity + 1")).Error
		}); err != nil {
			return err
		}
		if !redisEnqueued {
			UpdatePopularityCache(ctx, cs.cache, comment.VideoID, 1)
		}

	}

	return nil
}
func (cs *CommentService) Delete(ctx context.Context, commentid uint, accountID uint) error {
	comment, err := cs.repo.GetByID(ctx, commentid)
	if err != nil {
		return err
	}
	if comment == nil {
		return errors.New("comment not found")
	}
	if accountID != comment.AuthorID {
		return err
	}
	return cs.repo.DeleteComment(ctx, comment)
}
func (cs *CommentService) GetAll(ctx context.Context, id uint) ([]Comment, error) {
	ok, err := cs.VideoRepository.IsExist(ctx, id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("")

	}
	return cs.repo.GetAllComments(ctx, id)
}
