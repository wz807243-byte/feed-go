package video

import (
	"feedsystem_video_go/internal/middleware/jwt"

	"github.com/gin-gonic/gin"
)

type LikeHandler struct {
	service *LikeService
}

func NewLikeHandler(service *LikeService) *LikeHandler {
	return &LikeHandler{service: service}
}

func (lh *LikeHandler) Like(c *gin.Context) {
	var req LikeRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		return
	}
	if req.VideoID <= 0 {
		return
	}
	accountId, err := jwt.GetAccountID(c)
	if err != nil {
		return
	}
	like := &Like{
		VideoID:   req.VideoID,
		AccountID: accountId,
	}
	if err := lh.service.Like(c.Request.Context(), like); err != nil {
		return
	}
	

}

func (lh *LikeHandler) ListMyLikedVideos(c *gin.Context) {
	accountID, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
}