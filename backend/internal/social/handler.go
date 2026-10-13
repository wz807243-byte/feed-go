package social

import (
	"feedsystem_video_go/internal/account"
	"feedsystem_video_go/internal/apierror"
	"feedsystem_video_go/internal/middleware/jwt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type SocialHandler struct {
	service *SocialService
}

func NewSocialHandler(service *SocialService) *SocialHandler {
	return &SocialHandler{service: service}
}
func (h *SocialHandler) Follow(c *gin.Context) {
	var req FollowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Warn("invalid follow request body", zap.Error(err))
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if req.VloggerID <= 0 {
		zap.L().Warn("follow request missing vlogger ID")
		c.JSON(http.StatusBadRequest, gin.H{"error": "vlogger_id is required"})
		return
	}
	FollowerID, err := jwt.GetAccountID(c)
	if err != nil {
		zap.L().Warn("failed to get account ID for follow", zap.Error(err))
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	social := &Social{
		FollowerID: FollowerID,
		VloggerID:  req.VloggerID,
	}
	if err := h.service.Follow(c.Request.Context(), social); err != nil {
		zap.L().Error("failed to follow account", zap.Uint("follower_id", FollowerID), zap.Uint("vlogger_id", req.VloggerID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	zap.L().Info("account followed", zap.Uint("follower_id", FollowerID), zap.Uint("vlogger_id", req.VloggerID))
	c.JSON(http.StatusOK, gin.H{"message": "followed"})
}
func (h *SocialHandler) UnFoll(c *gin.Context) {
	var req FollowRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Warn("invalid unfollow request body", zap.Error(err))
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if req.VloggerID <= 0 {
		zap.L().Warn("unfollow request missing vlogger ID")
		c.JSON(http.StatusBadRequest, gin.H{"error": "vlogger_id is required"})
		return
	}
	FollwerID, err := jwt.GetAccountID(c)
	if err != nil {
		zap.L().Warn("failed to get account ID for unfollow", zap.Error(err))
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	social := &Social{
		FollowerID: FollwerID,
		VloggerID:  req.VloggerID,
	}
	if err := h.service.UnFollow(c.Request.Context(), social); err != nil {
		zap.L().Error("failed to unfollow account", zap.Uint("follower_id", FollwerID), zap.Uint("vlogger_id", req.VloggerID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	zap.L().Info("account unfollowed", zap.Uint("follower_id", FollwerID), zap.Uint("vlogger_id", req.VloggerID))
	c.JSON(http.StatusOK, gin.H{"message": "unfollowed"})
}
func (h *SocialHandler) GetallFoll(c *gin.Context) {
	var req FollowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Warn("invalid followers request body", zap.Error(err))
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	VloggerID := req.VloggerID
	if VloggerID == 0 {
		zap.L().Warn("followers request missing vlogger ID")
		c.JSON(http.StatusBadRequest, gin.H{"error": "vlogger_id is required"})
		return
	}
	var err error
	VloggerID, err = jwt.GetAccountID(c)
	if err != nil {
		zap.L().Warn("failed to get account ID for followers", zap.Error(err))
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	follwers, err := h.service.GetAllFollow(c.Request.Context(), VloggerID)
	if err != nil {
		zap.L().Error("failed to list followers", zap.Uint("vlogger_id", VloggerID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if follwers == nil {
		follwers = []*account.Account{}
	}
	c.JSON(200, follwers)
}
func (h *SocialHandler) GetVlogger(c *gin.Context) {
	var req GetAllVloggersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Warn("invalid followed accounts request body", zap.Error(err))
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	FollowerID := req.FollowerID
	if FollowerID <= 0 {
		zap.L().Warn("followed accounts request missing follower ID")
		c.JSON(http.StatusBadRequest, gin.H{"error": "follower_id is required"})
		return

	}
	var err error
	FollowerID, err = jwt.GetAccountID(c)
	if err != nil {
		zap.L().Warn("failed to get account ID for followed accounts", zap.Error(err))
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	vloggers, err := h.service.GetAllVloggers(c.Request.Context(), FollowerID)
	if err != nil {
		zap.L().Error("failed to list followed accounts", zap.Uint("follower_id", FollowerID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(vloggers) == 0 {
		vloggers = []*account.Account{}
	}
	c.JSON(200, vloggers)
}
func (h *SocialHandler) GetCounts(c *gin.Context) {
	accountID, err := jwt.GetAccountID(c)
	if err != nil {
		zap.L().Warn("failed to get account ID for social counts", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	followerCount, err := h.service.CountFollowers(c.Request.Context(), accountID)
	if err != nil {
		zap.L().Error("failed to count followers", zap.Uint("account_id", accountID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	vloggerCount, err := h.service.CountVloggers(c.Request.Context(), accountID)
	if err != nil {
		zap.L().Error("failed to count followed accounts", zap.Uint("account_id", accountID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, SocialCounts{FollowerCount: followerCount, VloggerCount: vloggerCount})
}
