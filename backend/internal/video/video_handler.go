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
		return
	}
	authID, err := jwt.GetAccountID(c)
	if err != nil {
		return
	}
	authName, err := jwt.GetUsername(c)
	if err != nil {
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
		return
	}
}
func (vh *VideoHandler) UploadVideo(c *gin.Context) {
	authID, err := jwt.GetAccountID(c)
	if err != nil {
		return
	}
	f, err := c.FormFile("file")
	if err != nil {
		return

	}
	const mexSize = 200 << 20
	if f.Size <= 0 || f.Size > mexSize {
		return
	}
	ext := strings.ToLower(filepath.Ext(f.Filename))
	if ext != ".mp4" {
		return
	}
	date := time.Now().Format("20260109")
	relDir := filepath.Join("videos", fmt.Sprintf("%d", authID), date)
	root := filepath.Join(".run", "uploads")
	absDri := filepath.Join(root, relDir)
	if err := os.MkdirAll(absDri, 0o755); err != nil {
		return
	}
	filename, err := randHex(16)
	if err != nil {
		return
	}
	filename = filename + ext
	absPath := filepath.Join(absDri, filename)

	if err := c.SaveUploadedFile(f, absPath); err != nil {
		return
	}
	urlPath := path.Join("/static", "videos", fmt.Sprintf("%d", authID), date, filename)

	c.JSON(http.StatusOK, gin.H{
		"url":       buildAbsoluteURL(c, urlPath),
		"cover_url": buildAbsoluteURL(c, urlPath),
	})
}

func (vh *VideoHandler) UploadCovers(c *gin.Context) {
	authID, err := jwt.GetAccountID(c)
	if err != nil {
		return
	}
	f, err := c.FormFile("file")
	if err != nil {
		return

	}
	const mexSize = 200 << 20
	if f.Size <= 0 || f.Size > mexSize {
		return
	}
	ext := strings.ToLower(filepath.Ext(f.Filename))
	switch ext {

	case ".jpg", ".jpeg", ".png", ".webp":
	default:

		return
	}

	date := time.Now().Format("20260109")
	relDir := filepath.Join("covers", fmt.Sprintf("%d", authID), date)
	root := filepath.Join(".run", "uploads")
	absDri := filepath.Join(root, relDir)
	if err := os.MkdirAll(absDri, 0o755); err != nil {
		return
	}
	filename, err := randHex(16)
	if err != nil {
		return
	}
	filename = filename + ext
	absPath := filepath.Join(absDri, filename)

	if err := c.SaveUploadedFile(f, absPath); err != nil {
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
		return
	}
	video, err := vh.videoservice.ListByAuthorID(c.Request.Context(), req.AuthorID)
	if err != nil {
		return
	}
	if video == nil {	
		video = []Video{}
	}
	c.JSON(200, video)

}
