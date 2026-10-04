package http

import (
	"feedsystem_video_go/internal/account"
	"feedsystem_video_go/internal/middleware/jwt"
	"feedsystem_video_go/internal/middleware/rabbitmq"
	rediscache "feedsystem_video_go/internal/middleware/redis"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func SetRouter(db *gorm.DB, cache *rediscache.Client, rmq *rabbitmq.RabbitMQ) *gin.Engine {
	r := gin.Default()
	if err := r.SetTrustedProxies(nil); err != nil {
		zap.L().Error("SetTrustedProxies failed", zap.Error(err))
	}
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})

	})
	r.Static("/static", "./.run/uploads")

	accountRepository := account.NewAccountRepository(db)
	accountService := account.NewAccountService(accountRepository)
	accountHandler := account.NewAcountHandler(accountService)
	accountGroup := r.Group("/account")
	accountGroup.POST("/login", accountHandler.Login)
	accountGroup.POST("/register", accountHandler.CreateAccount)
	accountGroup.POST("/findByUsername", accountHandler.FindByUsername)

	protectedAccount := accountGroup.Group("")
	protectedAccount.Use(jwt.JWTAuth(accountRepository, cache))
	protectedAccount.POST("/logout", accountHandler.Logout)
	protectedAccount.POST("/profile", accountHandler.UpdateProfile)
	protectedAccount.POST("/avatar", accountHandler.UploadAvatar)

	return r
}
