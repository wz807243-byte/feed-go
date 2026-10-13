package video

import (
	"feedsystem_video_go/internal/apierror"
	"feedsystem_video_go/internal/middleware/jwt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type LikeHandler struct {
	service *LikeService
}

func NewLikeHandler(service *LikeService) *LikeHandler {
	return &LikeHandler{service: service}
}

func (lh *LikeHandler) Like(c *gin.Context) {
	var req LikeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Warn("invalid like request body", zap.Error(err))
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if req.VideoID <= 0 {
		zap.L().Warn("like request missing video ID", zap.Uint("video_id", req.VideoID))
		c.JSON(http.StatusBadRequest, gin.H{"error": "video_id is required"})
		return
	}
	accountId, err := jwt.GetAccountID(c)
	if err != nil {
		zap.L().Warn("failed to get account ID for like", zap.Error(err))
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	like := &Like{
		VideoID:   req.VideoID,
		AccountID: accountId,
	}
	if err := lh.service.Like(c.Request.Context(), like, lh.service.cache); err != nil {
		zap.L().Error("failed to like video", zap.Uint("account_id", accountId), zap.Uint("video_id", req.VideoID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	zap.L().Info("video liked", zap.Uint("account_id", accountId), zap.Uint("video_id", req.VideoID))
	c.JSON(http.StatusOK, gin.H{"message": "video liked"})
}

func (lh *LikeHandler) Ulike(c *gin.Context) {

	var req LikeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Warn("invalid unlike request body", zap.Error(err))
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if req.VideoID <= 0 {
		zap.L().Warn("unlike request missing video ID", zap.Uint("video_id", req.VideoID))
		c.JSON(http.StatusBadRequest, gin.H{"error": "video_id is required"})
		return
	}
	accountId, err := jwt.GetAccountID(c)
	if err != nil {
		zap.L().Warn("failed to get account ID for unlike", zap.Error(err))
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	like := &Like{
		VideoID:   req.VideoID,
		AccountID: accountId,
	}
	if err := lh.service.ULike(c.Request.Context(), like); err != nil {
		zap.L().Error("failed to unlike video", zap.Uint("account_id", accountId), zap.Uint("video_id", req.VideoID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	zap.L().Info("video unliked", zap.Uint("account_id", accountId), zap.Uint("video_id", req.VideoID))
	c.JSON(http.StatusOK, gin.H{"message": "video unliked"})
}

func (lh *LikeHandler) IsLiked(c *gin.Context) {
	var req LikeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Warn("invalid is-liked request body", zap.Error(err))
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if req.VideoID <= 0 {
		zap.L().Warn("is-liked request missing video ID", zap.Uint("video_id", req.VideoID))
		c.JSON(400, gin.H{"error": "video_id is required"})
		return
	}

	accountID, err := jwt.GetAccountID(c)
	if err != nil {
		zap.L().Warn("failed to get account ID for is-liked request", zap.Error(err))
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	isLiked, err := lh.service.IsLiked(c.Request.Context(), req.VideoID, accountID)
	if err != nil {
		zap.L().Error("failed to check video like", zap.Uint("account_id", accountID), zap.Uint("video_id", req.VideoID), zap.Error(err))
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"is_liked": isLiked})
}

func (lh *LikeHandler) ListMyLikedVideos(c *gin.Context) {
	accountID, err := jwt.GetAccountID(c)
	if err != nil {
		zap.L().Warn("failed to get account ID for liked videos", zap.Error(err))
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	videos, err := lh.service.ListLikedVideos(c.Request.Context(), accountID)
	if err != nil {
		zap.L().Error("failed to list liked videos", zap.Uint("account_id", accountID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if videos == nil {
		videos = []Video{}
	}
	c.JSON(http.StatusOK, videos)
}
