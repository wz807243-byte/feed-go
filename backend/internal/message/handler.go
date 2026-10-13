package message

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"feedsystem_video_go/internal/apierror"
	"feedsystem_video_go/internal/middleware/jwt"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }
type Service struct{ repo *Repository }
type Handler struct{ service *Service }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }
func NewService(repo *Repository) *Service  { return &Service{repo: repo} }
func NewHandler(service *Service) *Handler  { return &Handler{service: service} }

func (r *Repository) AutoMigrate(ctx context.Context) error {
	return r.db.WithContext(ctx).AutoMigrate(&Message{})
}
func (r *Repository) send(ctx context.Context, m *Message) error {
	m.Content = strings.TrimSpace(m.Content)
	if m.Content == "" {
		return errors.New("content is nil")
	}
	m.CreatedAt = time.Now()
	return r.db.WithContext(ctx).Create(m).Error
}
func (r *Repository) List(ctx context.Context, userID, peerID uint, limit int) ([]Message, error) {
	var msgs []Message
	err := r.db.WithContext(ctx).
		Where("(from_id = ? AND to_id = ?) OR (from_id = ? AND to_id = ?)", userID, peerID, peerID, userID).
		Order("created_at desc").
		Limit(limit).
		Find(&msgs).Error
	return msgs, err
}

func (s *Service) Send(ctx context.Context, fromID, toID uint, content string) error {
	return s.repo.send(ctx, &Message{
		FromID:  fromID,
		ToID:    toID,
		Content: content,
	})
}

func (s *Service) List(ctx context.Context, userID, peerID uint) ([]Message, error) {
	const limit = 100
	return s.repo.List(ctx, userID, peerID, limit)
}

func (h *Handler) Send(c *gin.Context) {
	var req SendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if req.ToID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "to_id is required"})
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content is required"})
		return
	}

	accountID, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if err := h.service.Send(c.Request.Context(), accountID, req.ToID, req.Content); err != nil {
		zap.L().Error("failed to send message", zap.Uint("from_id", accountID), zap.Uint("to_id", req.ToID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send message"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "message sent"})
}

func (h *Handler) List(c *gin.Context) {
	var req ListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if req.PeerID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "peer_id is required"})
		return
	}

	accountID, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	messages, err := h.service.List(c.Request.Context(), accountID, req.PeerID)
	if err != nil {
		zap.L().Error("failed to list messages", zap.Uint("account_id", accountID), zap.Uint("peer_id", req.PeerID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list messages"})
		return
	}
	if messages == nil {
		messages = []Message{}
	}
	c.JSON(http.StatusOK, ListResponse{Messages: messages})
}
