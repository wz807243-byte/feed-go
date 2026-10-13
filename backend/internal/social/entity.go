package social

type Social struct {
	ID         uint `gorm:"primaryKey"`
	FollowerID uint `gorm:"not null;index:idx_social_follower;uniqueIndex:idx_social_follower_vlogger"`
	VloggerID  uint `gorm:"not null;index:idx_social_vlogger;uniqueIndex:idx_social_follower_vlogger"`
}

type FollowRequest struct {
	VloggerID uint `json:"vlogger_id"`
}

type GetAllVloggersRequest struct {
	FollowerID uint `json:"follower_id"`
}
type SocialCounts struct {
	FollowerCount int64 `json:"follower_count"`
	VloggerCount  int64 `json:"vlogger_count"`
}
