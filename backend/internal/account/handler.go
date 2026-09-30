package account

import (
	"errors"
	"feedsystem_video_go/internal/apierror"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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
