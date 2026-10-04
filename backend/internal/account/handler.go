package account

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"feedsystem_video_go/internal/apierror"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AccountHandler struct {
	accountService *AccountService
}

func NewAcountHandler(accountService *AccountService) *AccountHandler {
	return &AccountHandler{accountService: accountService}
}
func (h *AccountHandler) CreateAccount(c *gin.Context) {
	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {

		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if err := h.accountService.CreateAccount(c.Request.Context(), &Account{
		Username: req.Username,
		Password: req.Password,
	}); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "account created"})
}
func (h *AccountHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"account error": err.Error()})
		return

	}
	account, err := h.accountService.FindByUsername(c.Request.Context(), req.Username)
	if err != nil {
		c.JSON(500, gin.H{"account error": err.Error()})
		return

	}
	accessToken, refreshToken, err := h.accountService.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		c.JSON(500, gin.H{"accessToken, refreshToken error": err.Error()})
		return
	}
	c.JSON(200, LoginResponse{Token: accessToken, RefreshToken: refreshToken, AccountID: account.ID, Username: account.Username})

}
func (h *AccountHandler) Logout(c *gin.Context) {
	accountID, err := getAccountID(c)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		zap.L().Error("")
		return
	}
	if err := h.accountService.Logout(c.Request.Context(), accountID); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "account logged out"})
}
func (h *AccountHandler) FindByUsername(c *gin.Context) {
	var req FindByUsernameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return
	}
	userNaem, err := h.accountService.FindByUsername(c.Request.Context(), req.Username)
	if err != nil {
		return
	}
	c.JSON(200, gin.H{"Usernaem": userNaem})
}
func (h *AccountHandler) FindByID(c *gin.Context) {
	var req FindByIDRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		return
	}
	userId, err := h.accountService.FindByID(c.Request.Context(), req.ID)
	if err != nil {
		return
	}
	c.JSON(200, userId)
}

func (h *AccountHandler) UpdateProfile(c *gin.Context) {
	accountID, err := getAccountID(c)
	if err != nil {
		return
	}
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{})
		return
	}
	if err := h.accountService.UpdateProfile(c.Request.Context(), accountID, &req); err != nil {
		return
	}

}
func (h *AccountHandler) UploadAvatar(c *gin.Context) {
	accountID, err := getAccountID(c)
	if err != nil {
		return
	}
	f, err := c.FormFile("file")
	if err != nil {
		return
	}
	const maxSize = 10 << 20
	if f.Size <= 0 || f.Size > maxSize {
		return
	}
	ext := strings.ToLower(filepath.Ext(f.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "only .jpg/.jpeg/.png/.webp allowed"})
		return
	}
	dir := filepath.Join(".run", "uploads", "avatars", strconv.FormatUint(uint64(accountID), 10))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	filename, err := randHex(16)
	if err != nil {
		return

	}
	filename = filename + ext
	adsPaht := filepath.Join(dir, filename)
	if err := c.SaveUploadedFile(f, adsPaht); err != nil {
		return
	}
	urlPath := path.Join("/static", "avatars", strconv.FormatUint(uint64(accountID), 10), filename)
	avatarURL := buildAbsoluteURL(c, urlPath)
	if err := h.accountService.UpdateAvatar(c.Request.Context(), accountID, avatarURL); err != nil {
		return
	}
	c.JSON(200, gin.H{"avatar_url": avatarURL})
}
func (h *AccountHandler) Rename(c *gin.Context) {
	var req RenameRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		return
	}
	account, err := getAccountID(c)
	if err != nil {
		return
	}
	token, err := h.accountService.Rename(c.Request.Context(), account, req.NewUsername)
	if err != nil {
		if errors.Is(err, ErrNewUsernameRequired) {
			c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, ErrUsernameTaken) {
			c.JSON(409, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"error": "account not found"})
			return
		}
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"token": token})
}

func (h *AccountHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		return
	}
	if err := h.accountService.ChangePassword(c.Request.Context(), req.NewPassword, req.OldPassword, req.NewPassword); err != nil {
		return
	}

}

func (h *AccountHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		return
	}
	accountID, userName, Newtoken, err := h.accountService.RefreshAccessToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		return
	}
	c.JSON(200, gin.H{
		"account_id": accountID,
		"username":   userName,
		"token":      Newtoken,
	})
	return
}
func getAccountID(c *gin.Context) (uint, error) {
	value, exists := c.Get("accountID")
	if !exists {
		return 0, errors.New("accountID not found")
	}
	id, ok := value.(uint)
	if !ok {
		return 0, errors.New("accountID has invalid type")
	}
	return id, nil
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
