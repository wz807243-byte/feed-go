package feed

import (
	"feedsystem_video_go/internal/apierror"
	"feedsystem_video_go/internal/middleware/jwt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type FeedHandler struct {
	service *FeedService
}

func NewFeedHandler(service *FeedService) *FeedHandler {
	return &FeedHandler{service: service}
}
func (h *FeedHandler) ListLatest(c *gin.Context) {
	var req ListLatestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Warn("invalid latest feed request body", zap.Error(err))
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = 10
	}
	var LatestTime time.Time
	if req.LatestTime > 0 {
		LatestTime = time.UnixMilli(req.LatestTime)
	}
	viewerAccountID, err := jwt.GetAccountID(c)
	if err != nil {
		viewerAccountID = 0 // SoftJWTAuth permits anonymous feed requests.
	}
	feedItems, err := h.service.ListLatest(c.Request.Context(), req.Limit, LatestTime, viewerAccountID)
	if err != nil {
		zap.L().Error("failed to list latest feed", zap.Uint("account_id", viewerAccountID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list latest feed"})
		return
	}
	if feedItems.VideoList == nil {
		feedItems.VideoList = []FeedVideoItem{}
	}
	c.JSON(200, feedItems)
}
func (h *FeedHandler) LekiListCount(c *gin.Context) {
	var req ListLikesCountRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		zap.L().Warn("invalid likes-count feed request body", zap.Error(err))
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if req.Limit > 50 || req.Limit <= 0 {
		req.Limit = 10
	}
	var cursor *LikesCountCursor
	if req.IDBefore != nil || req.LikesCountBefore != nil {
		if req.IDBefore == nil || req.LikesCountBefore == nil {
			zap.L().Warn("incomplete likes-count feed cursor")
			c.JSON(400, gin.H{"error": "both id_before and likes_count_before are required"})
			return
		}
		likesCountBefore := *req.LikesCountBefore
		idBefore := *req.IDBefore

		if likesCountBefore < 0 {
			zap.L().Warn("invalid likes-count feed cursor", zap.Int64("likes_count_before", likesCountBefore))
			c.JSON(400, gin.H{"error": "invalid cursor: likes_count_before must be >= 0"})
			return
		}
		if idBefore == 0 {
			zap.L().Warn("invalid likes-count feed cursor", zap.Uint("id_before", idBefore))

			c.JSON(400, gin.H{"error": "invalid cursor: id_before must be > 0"})
			return

		}

		cursor = &LikesCountCursor{
			LikesCount: likesCountBefore,
			ID:         idBefore,
		}
	}

	accountID, err := jwt.GetAccountID(c)
	if err != nil {
		accountID = 0
	}
	feedItems, err := h.service.ListLikesCount(c.Request.Context(), req.Limit, cursor, accountID)
	if err != nil {
		zap.L().Error("failed to list feed by likes count", zap.Uint("account_id", accountID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list feed by likes count"})
		return
	}
	c.JSON(200, feedItems)
}

func (h *FeedHandler) ListByFollowing(c *gin.Context) {
	var req ListByFollowingRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		zap.L().Warn("invalid following feed request body", zap.Error(err))
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = 10
	}
	viewerAccountID, err := jwt.GetAccountID(c)
	if err != nil {
		zap.L().Warn("failed to get account ID for following feed", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var latestTime time.Time
	if req.LatestTime > 0 {
		latestTime = time.Unix(req.LatestTime, 0)
	}
	feedItems, err := h.service.ListByFollowing(c.Request.Context(), req.Limit, latestTime, viewerAccountID)
	if err != nil {
		zap.L().Error("failed to list following feed", zap.Uint("account_id", viewerAccountID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list following feed"})
		return
	}
	c.JSON(200, feedItems)
}

func (f *FeedHandler) ListByPopularity(c *gin.Context) {
	var req ListByPopularityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = 10
	}
	viewerAccountID, err := jwt.GetAccountID(c)
	if err != nil {
		viewerAccountID = 0
	}
	var latestPopularity int64
	var latestBefore time.Time
	var latestIDBefore uint
	anyCursor := !req.LatestBefore.IsZero() || req.LatestIDBefore != nil
	if anyCursor {
		if req.LatestBefore.IsZero() || req.LatestIDBefore == nil || *req.LatestIDBefore == 0 {
			c.JSON(400, gin.H{"error": "latest_before and latest_id_before must be provided together"})
			return
		}
		latestPopularity = req.LatestPopularity
		latestBefore = req.LatestBefore
		latestIDBefore = *req.LatestIDBefore
	}
	resp, err := f.service.ListByPopularity(
		c.Request.Context(),
		req.Limit,
		req.AsOf,
		req.Offset,
		viewerAccountID,
		latestPopularity,
		latestBefore,
		latestIDBefore,
	)
	if err != nil {
		zap.L().Error("failed to list feed by popularity", zap.Uint("account_id", viewerAccountID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list feed by popularity"})
		return
	}
	if resp.VideoList == nil {
		resp.VideoList = []FeedVideoItem{}
	}
	c.JSON(200, resp)
}

func (h *FeedHandler) ListByTag(c *gin.Context) {
	var req struct {
		TagName string `json:"tag_name"`
		Limit   int    `json:"limit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if req.TagName == "" {
		c.JSON(400, gin.H{"error": "tag_name is required"})
		return
	}
	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = 10
	}
	viewerAccountID, _ := jwt.GetAccountID(c)
	items, err := h.service.ListByTag(c.Request.Context(), req.TagName, req.Limit, viewerAccountID)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if items == nil {
		items = []FeedVideoItem{}
	}

	c.JSON(200, items)
}
