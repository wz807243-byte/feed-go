package video

import (
	"feedsystem_video_go/internal/account"
	"feedsystem_video_go/internal/apierror"
	"feedsystem_video_go/internal/middleware/jwt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type CommentHandler struct {
	service        *CommentService
	accountService *account.AccountService
}

func NewCommentHandler(service *CommentService,
	accountService *account.AccountService) *CommentHandler {
	return &CommentHandler{service: service, accountService: accountService}
}
func (ch *CommentHandler) PublishComment(c *gin.Context) {
	var req PublishCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Warn("invalid publish comment request body", zap.Error(err))
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if req.Content == "" {
		zap.L().Warn("publish comment request missing content")
		c.JSON(http.StatusBadRequest, gin.H{"error": "content is required"})
		return
	}
	if req.VideoID <= 0 {
		zap.L().Warn("publish comment request missing video ID", zap.Uint("video_id", req.VideoID))
		c.JSON(http.StatusBadRequest, gin.H{"error": "video_id is required"})
		return
	}
	commentuserID, err := jwt.GetAccountID(c)
	if err != nil {
		zap.L().Warn("failed to get account ID for comment publish", zap.Error(err))
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	userNeme, err := ch.accountService.FindByID(c.Request.Context(), commentuserID)
	if err != nil {
		zap.L().Error("failed to find comment author", zap.Uint("account_id", commentuserID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	comment := &Comment{
		Username: userNeme.Username,
		VideoID:  req.VideoID,
		Content:  req.Content,
		AuthorID: commentuserID,
	}
	if err := ch.service.Publish(c.Request.Context(), comment); err != nil {
		zap.L().Error("failed to publish comment", zap.Uint("account_id", commentuserID), zap.Uint("video_id", req.VideoID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	zap.L().Info("comment published", zap.Uint("account_id", commentuserID), zap.Uint("video_id", req.VideoID))
	c.JSON(http.StatusOK, gin.H{"message": "comment published", "comment": comment})
}
func (ch *CommentHandler) DeletComment(c *gin.Context) {
	var req DeleteCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Warn("invalid delete comment request body", zap.Error(err))
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	accountID, err := jwt.GetAccountID(c)
	if err != nil {
		zap.L().Warn("failed to get account ID for comment deletion", zap.Error(err))
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if req.CommentID <= 0 {
		zap.L().Warn("delete comment request missing comment ID", zap.Uint("comment_id", req.CommentID))
		c.JSON(http.StatusBadRequest, gin.H{"error": "comment_id is required"})
		return
	}
	if err := ch.service.Delete(c.Request.Context(), req.CommentID, accountID); err != nil {
		zap.L().Error("failed to delete comment", zap.Uint("account_id", accountID), zap.Uint("comment_id", req.CommentID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	zap.L().Info("comment deleted", zap.Uint("account_id", accountID), zap.Uint("comment_id", req.CommentID))
	c.JSON(http.StatusOK, gin.H{"message": "comment deleted"})
}
func (ch *CommentHandler) AllComment(c *gin.Context) {
	var req GetAllCommentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Warn("invalid list comments request body", zap.Error(err))
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if req.VideoID == 0 {
		zap.L().Warn("list comments request missing video ID")
		c.JSON(http.StatusBadRequest, gin.H{"error": "video_id is required"})
		return
	}
	comment, err := ch.service.GetAll(c.Request.Context(), req.VideoID)
	if err != nil {
		zap.L().Error("failed to list comments", zap.Uint("video_id", req.VideoID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if comment == nil {
		comment = []Comment{}

	}

	c.JSON(http.StatusOK, comment)
}
