package social

import (
	"context"
	"errors"
	"feedsystem_video_go/internal/account"
	rediscache "feedsystem_video_go/internal/middleware/redis"
)

type SocialService struct {
	repo        *SocialRepository
	accountrepo *account.AccountRepository

	cache *rediscache.Client
}

func NewSocialService(repo *SocialRepository, accountrepo *account.AccountRepository, cache *rediscache.Client) *SocialService {
	return &SocialService{repo: repo, accountrepo: accountrepo, cache: cache}
}
func (s *SocialService) Follow(ctx context.Context, social *Social) error {
	_, err := s.accountrepo.FindByID(ctx, social.FollowerID)
	if err != nil {
		return err
	}
	_, err = s.accountrepo.FindByID(ctx, social.VloggerID)
	if err != nil {
		return err
	}
	isFoll, _ := s.repo.IsFollowed(ctx, social)
	if isFoll {
		return nil
	}
	if err := s.repo.Foll(ctx, social); err != nil {
		return err
	}
	s.invalidateFollowingFeedCache(context.Background(), social.FollowerID)
	return nil

}
func (s *SocialService) invalidateFollowingFeedCache(ctx context.Context, accountID uint) {
	if s.cache == nil {
		return
	}
	pattern := s.cache.Key("feed:listByFollowing:*:accountID=%d:*", accountID)
	if err := s.cache.DelByPattern(ctx, pattern); err != nil {
		return
	}
}
func (s *SocialService) UnFollow(ctx context.Context, social *Social) error {
	_, err := s.accountrepo.FindByID(ctx, social.FollowerID)
	if err != nil {
		return err
	}
	_, err = s.accountrepo.FindByID(ctx, social.VloggerID)
	if err != nil {
		return err
	}
	isFoll, err := s.repo.IsFollowed(ctx, social)
	if err != nil {
		return err
	}
	if !isFoll {
		return errors.New("NO FOlls")
	}
	if err := s.repo.UnFoll(ctx, social); err != nil {
		return err
	}
	s.invalidateFollowingFeedCache(context.Background(), social.FollowerID)
	return nil
}
func (s *SocialService) GetAllFollow(ctx context.Context, id uint) ([]*account.Account, error) {
	_, err := s.accountrepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.repo.GetAllFoll(ctx, id)
}

func (s *SocialService) GetAllVloggers(ctx context.Context, id uint) ([]*account.Account, error) {
	_, err := s.accountrepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.repo.GetAllVloggers(ctx, id)
}
func (s *SocialService) CountFollowers(ctx context.Context, vloggerID uint) (int64, error) {
	return s.repo.CountFollowers(ctx, vloggerID)
}
func (s *SocialService) CountVloggers(ctx context.Context, vloggerID uint) (int64, error) {
	return s.repo.CountVloggers(ctx, vloggerID)
}
