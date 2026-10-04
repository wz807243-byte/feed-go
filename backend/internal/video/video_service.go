package video

import (
	"context"
	"errors"
	"strings"
)

type VideoService struct {
	videorepo *VideoRepository
}

func NewVideoService(repo *VideoRepository) *VideoService {
	return &VideoService{videorepo: repo}
}
func (vs *VideoService) Publish(ctx context.Context, video *Video) error {
	if video == nil {
		return errors.New("video is nil	")
	}
	video.Title = strings.TrimSpace(video.Title)
	video.PlayURL = strings.TrimSpace(video.PlayURL)
	video.CoverURL = strings.TrimSpace(video.CoverURL)
	if video.Title == "" {
		return errors.New("Title is nil	")
	}
	if video.PlayURL == "" {
		return errors.New("PlayURL is nil	")
	}
	if video.CoverURL == "" {
		return errors.New("CoverURL is nil	")
	}
	if err := vs.videorepo.CreateVideo(ctx, video); err != nil {
		return err
	}
	return nil
}
func (vs *VideoService) ListByAuthorID(ctx context.Context, id uint) ([]Video, error) {
	if id == 0 {
		return nil, errors.New("id is nil")
	}
	video, err := vs.videorepo.ListByAuthorID(ctx, int64(id))
	if err != nil {
		return nil, err
	}
	return video, err
}
