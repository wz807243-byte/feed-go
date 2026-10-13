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
		zap.L().Error("failed to create account", zap.String("username", req.Username), zap.Error(err))
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
		zap.L().Error("failed to find account for login", zap.String("username", req.Username), zap.Error(err))
		c.JSON(500, gin.H{"account error": err.Error()})
		return

	}
	accessToken, refreshToken, err := h.accountService.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		zap.L().Error("failed to issue account login tokens", zap.Uint("account_id", account.ID), zap.Error(err))
		c.JSON(500, gin.H{"accessToken, refreshToken error": err.Error()})
		return
	}
	c.JSON(200, LoginResponse{Token: accessToken, RefreshToken: refreshToken, AccountID: account.ID, Username: account.Username})

}
func (h *AccountHandler) Logout(c *gin.Context) {
	accountID, err := getAccountID(c)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})

		zap.L().Warn("account ID missing from logout context", zap.Error(err))
		return
	}
	if err := h.accountService.Logout(c.Request.Context(), accountID); err != nil {
		zap.L().Error("failed to log out account", zap.Uint("account_id", accountID), zap.Error(err))
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "account logged out"})
}
func (h *AccountHandler) FindByUsername(c *gin.Context) {
	var req FindByUsernameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userNaem, err := h.accountService.FindByUsername(c.Request.Context(), req.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
			return
		}
		zap.L().Error("failed to find account by username", zap.String("username", req.Username), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find account"})
		return
	}
	c.JSON(200, gin.H{"Usernaem": userNaem})
}
func (h *AccountHandler) FindByID(c *gin.Context) {
	var req FindByIDRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userId, err := h.accountService.FindByID(c.Request.Context(), req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
			return
		}
		zap.L().Error("failed to find account by ID", zap.Uint("account_id", req.ID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find account"})
		return
	}
	c.JSON(200, userId)
}

func (h *AccountHandler) UpdateProfile(c *gin.Context) {
	accountID, err := getAccountID(c)
	if err != nil {
		zap.L().Warn("account ID missing from profile context", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if err := h.accountService.UpdateProfile(c.Request.Context(), accountID, &req); err != nil {
		zap.L().Error("failed to update account profile", zap.Uint("account_id", accountID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "profile updated"})

}
func (h *AccountHandler) UploadAvatar(c *gin.Context) {
	accountID, err := getAccountID(c)
	if err != nil {
		zap.L().Warn("account ID missing from avatar upload context", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	f, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	const maxSize = 10 << 20
	if f.Size <= 0 || f.Size > maxSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file size must be between 1 byte and 10 MB"})
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
		zap.L().Error("failed to create avatar directory", zap.Uint("account_id", accountID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	filename, err := randHex(16)
	if err != nil {
		zap.L().Error("failed to generate avatar filename", zap.Uint("account_id", accountID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "avatar upload failed"})
		return
	}
	filename = filename + ext
	adsPaht := filepath.Join(dir, filename)
	if err := c.SaveUploadedFile(f, adsPaht); err != nil {
		zap.L().Error("failed to save uploaded avatar", zap.Uint("account_id", accountID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "avatar upload failed"})
		return
	}
	urlPath := path.Join("/static", "avatars", strconv.FormatUint(uint64(accountID), 10), filename)
	avatarURL := buildAbsoluteURL(c, urlPath)
	if err := h.accountService.UpdateAvatar(c.Request.Context(), accountID, avatarURL); err != nil {
		zap.L().Error("failed to store avatar URL", zap.Uint("account_id", accountID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "avatar upload failed"})
		return
	}
	c.JSON(200, gin.H{"avatar_url": avatarURL})
}
func (h *AccountHandler) Rename(c *gin.Context) {
	var req RenameRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	account, err := getAccountID(c)
	if err != nil {
		zap.L().Warn("account ID missing from rename context", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
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
		zap.L().Error("failed to rename account", zap.Uint("account_id", account), zap.Error(err))
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"Ntoken": token})
}

func (h *AccountHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if err := h.accountService.ChangePassword(c.Request.Context(), req.Username, req.OldPassword, req.NewPassword); err != nil {
		zap.L().Error("failed to change account password", zap.String("username", req.Username), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "successfully password changed"})
}

func (h *AccountHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	accountID, userName, Newtoken, err := h.accountService.RefreshAccessToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || req.RefreshToken == "" || err.Error() == "invalid refresh token" || err.Error() == "refresh token is empty" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
			return
		}
		zap.L().Error("failed to refresh account access token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
