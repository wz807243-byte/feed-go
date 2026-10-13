package video

import (
	"crypto/rand"
	"encoding/hex"
	"feedsystem_video_go/internal/account"
	"feedsystem_video_go/internal/middleware/jwt"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type VideoHandler struct {
	videoservice   *VideoService
	accountService *account.AccountService
}

func NewVideoHandler(videoservice *VideoService, accountService *account.AccountService) *VideoHandler {
	return &VideoHandler{videoservice: videoservice, accountService: accountService}
}
func (vh *VideoHandler) PublishVideo(c *gin.Context) {
	var req *PublishVideoRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	authID, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	authName, err := jwt.GetUsername(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	viDeo := &Video{
		AuthorID:    authID,
		Username:    authName,
		Title:       req.Title,
		Description: req.Description,
		PlayURL:     req.PlayURL,
		CoverURL:    req.CoverURL,
		CreateTime:  time.Now(),
	}

	if err := vh.videoservice.Publish(c.Request.Context(), viDeo); err != nil {
		zap.L().Error("failed to publish video", zap.Uint("account_id", authID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to publish video"})
		return
	}
	c.JSON(200, gin.H{"m": "ok"})
}
func (vh *VideoHandler) UploadVideo(c *gin.Context) {
	authID, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	f, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	const mexSize = 200 << 20
	if f.Size <= 0 || f.Size > mexSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file size must be between 1 byte and 200 MB"})
		return
	}
	ext := strings.ToLower(filepath.Ext(f.Filename))
	if ext != ".mp4" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only .mp4 files are allowed"})
		return
	}
	date := time.Now().Format("20260109")
	relDir := filepath.Join("videos", fmt.Sprintf("%d", authID), date)
	root := filepath.Join(".run", "uploads")
	absDri := filepath.Join(root, relDir)
	if err := os.MkdirAll(absDri, 0o755); err != nil {
		zap.L().Error("failed to create video upload directory", zap.Uint("account_id", authID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "video upload failed"})
		return
	}
	filename, err := randHex(16)
	if err != nil {
		zap.L().Error("failed to generate video filename", zap.Uint("account_id", authID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "video upload failed"})
		return
	}
	filename = filename + ext
	absPath := filepath.Join(absDri, filename)

	if err := c.SaveUploadedFile(f, absPath); err != nil {
		zap.L().Error("failed to save uploaded video", zap.Uint("account_id", authID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "video upload failed"})
		return
	}
	urlPath := path.Join("/static", "videos", fmt.Sprintf("%d", authID), date, filename)

	c.JSON(http.StatusOK, gin.H{
		"url":      buildAbsoluteURL(c, urlPath),
		"play_url": buildAbsoluteURL(c, urlPath),
	})
}

func (vh *VideoHandler) UploadCovers(c *gin.Context) {
	authID, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	f, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	const mexSize = 200 << 20
	if f.Size <= 0 || f.Size > mexSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file size must be between 1 byte and 200 MB"})
		return
	}
	ext := strings.ToLower(filepath.Ext(f.Filename))
	switch ext {

	case ".jpg", ".jpeg", ".png", ".webp":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "only .jpg/.jpeg/.png/.webp files are allowed"})
		return
	}

	date := time.Now().Format("20260109")
	relDir := filepath.Join("covers", fmt.Sprintf("%d", authID), date)
	root := filepath.Join(".run", "uploads")
	absDri := filepath.Join(root, relDir)
	if err := os.MkdirAll(absDri, 0o755); err != nil {
		zap.L().Error("failed to create cover upload directory", zap.Uint("account_id", authID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cover upload failed"})
		return
	}
	filename, err := randHex(16)
	if err != nil {
		zap.L().Error("failed to generate cover filename", zap.Uint("account_id", authID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cover upload failed"})
		return
	}
	filename = filename + ext
	absPath := filepath.Join(absDri, filename)

	if err := c.SaveUploadedFile(f, absPath); err != nil {
		zap.L().Error("failed to save uploaded cover", zap.Uint("account_id", authID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cover upload failed"})
		return
	}
	urlPath := path.Join("/static", "covers", fmt.Sprintf("%d", authID), date, filename)

	c.JSON(http.StatusOK, gin.H{
		"url":       buildAbsoluteURL(c, urlPath),
		"cover_url": buildAbsoluteURL(c, urlPath),
	})
}

func randHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand.Read: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func buildAbsoluteURL(c *gin.Context, p string) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if xf := c.GetHeader("X-Forwarded-Proto"); xf != "" {
		scheme = xf
	}
	return fmt.Sprintf("%s://%s%s", scheme, c.Request.Host, p)
}
func (vh *VideoHandler) ListByAuthorID(c *gin.Context) {
	var req ListByAuthorIDRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if req.AuthorID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "author_id is required"})
		return
	}
	video, err := vh.videoservice.ListByAuthorID(c.Request.Context(), req.AuthorID)
	if err != nil {
		zap.L().Error("failed to list videos by author", zap.Uint("author_id", req.AuthorID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list videos"})
		return
	}
	if video == nil {
		video = []Video{}
	}
	c.JSON(200, video)

}

// func (vh *VideoHandler) GetDetail(c *gin.Context) {
// 	var req GetDetailRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {

// 		return
// 	}
// 	video, err := vh.videoservice.GetDetail(c.Request.Context(), req.ID)
// 	if err != nil {
// 		return
// 	}
// }
