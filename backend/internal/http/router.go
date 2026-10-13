package http

import (
	"feedsystem_video_go/internal/account"
	"feedsystem_video_go/internal/feed"
	"feedsystem_video_go/internal/message"
	"feedsystem_video_go/internal/middleware/jwt"
	"feedsystem_video_go/internal/middleware/rabbitmq"
	rediscache "feedsystem_video_go/internal/middleware/redis"
	"feedsystem_video_go/internal/social"
	"feedsystem_video_go/internal/video"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func SetRouter(db *gorm.DB, cache *rediscache.Client, rmq *rabbitmq.RabbitMQ) *gin.Engine {
	r := gin.Default()
	if err := r.SetTrustedProxies(nil); err != nil {
		zap.L().Error("SetTrustedProxies failed", zap.Error(err))
	}
	r.GET("/OK", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "你好"})

	})
	r.Static("/static", "./.run/uploads")
	// account

	accountRepository := account.NewAccountRepository(db)
	accountService := account.NewAccountService(accountRepository, cache)
	accountHandler := account.NewAcountHandler(accountService)
	accountGroup := r.Group("/account")

	accountGroup.POST("/login", accountHandler.Login)
	accountGroup.POST("/register", accountHandler.CreateAccount)
	accountGroup.POST("/findByUsername", accountHandler.FindByUsername)
	accountGroup.POST("/findByID", accountHandler.FindByID)

	protectedAccount := accountGroup.Group("")
	protectedAccount.Use(jwt.JWTAuth(accountRepository, cache))

	protectedAccount.POST("/logout", accountHandler.Logout)
	protectedAccount.POST("/profile", accountHandler.UpdateProfile)
	protectedAccount.POST("/avatar", accountHandler.UploadAvatar)
	protectedAccount.POST("/rename", accountHandler.Rename)
	accountGroup.POST("/refresh", accountHandler.Refresh)
	accountGroup.POST("/changepassword", accountHandler.ChangePassword)

	// video

	videoRepository := video.NewVideoRepositor(db)

	videoService := video.NewVideoService(videoRepository, cache)
	videoHandler := video.NewVideoHandler(videoService, accountService)
	videoGroup := r.Group("/video")

	videoGroup.POST("/listByAuthorID", videoHandler.ListByAuthorID)

	protectedVideoGroup := videoGroup.Group("")
	protectedVideoGroup.Use(jwt.JWTAuth(accountRepository, cache))
	protectedVideoGroup.POST("/uploadVideo", videoHandler.UploadVideo)
	protectedVideoGroup.POST("/uploadCover", videoHandler.UploadCovers)
	protectedVideoGroup.POST("/publish", videoHandler.PublishVideo)

	// like

	likeRepository := video.NewLiekRepostiory(db)
	likeService := video.NewLikeService(likeRepository, videoRepository, cache)
	likeHandler := video.NewLikeHandler(likeService)
	likeGroup := r.Group("/like")
	protectedLikeGroup := likeGroup.Group("")
	protectedLikeGroup.Use(jwt.JWTAuth(accountRepository, cache))

	protectedLikeGroup.POST("/like", likeHandler.Like)
	protectedLikeGroup.POST("/unlike", likeHandler.Ulike)
	protectedLikeGroup.POST("/isLiked", likeHandler.IsLiked)
	protectedLikeGroup.POST("/listMyLikedVideos", likeHandler.ListMyLikedVideos)

	// comment

	commentRepository := video.NewCommentRepository(db)
	commentService := video.NewCommentService(commentRepository, videoRepository, cache)
	commentHandler := video.NewCommentHandler(commentService, accountService)
	commentGroup := r.Group("/comment")

	commentGroup.POST("/listAll", commentHandler.AllComment)

	protectedCommentGroup := commentGroup.Group("")
	protectedCommentGroup.Use(jwt.JWTAuth(accountRepository, cache))

	protectedCommentGroup.POST("/publish", commentHandler.PublishComment)
	protectedCommentGroup.POST("/delete", commentHandler.DeletComment)

	// social
	socialRepository := social.NewSocialRepository(db)
	socialService := social.NewSocialService(socialRepository, accountRepository, cache)
	socialHandler := social.NewSocialHandler(socialService)
	socialGroup := r.Group("/social")
	protectedSocialGroup := socialGroup.Group("")
	protectedSocialGroup.Use(jwt.JWTAuth(accountRepository, cache))

	protectedSocialGroup.POST("/follow", socialHandler.Follow)
	protectedSocialGroup.POST("/unfollow", socialHandler.UnFoll)
	protectedSocialGroup.POST("/getAllFollowers", socialHandler.GetallFoll)
	protectedSocialGroup.POST("/getAllVloggers", socialHandler.GetVlogger)
	protectedSocialGroup.POST("/getCounts", socialHandler.GetCounts)
	// feed
	feedRepository := feed.NewFeedRepository(db)
	feedService := feed.NewFeedService(feedRepository, likeRepository, cache)
	feedHandler := feed.NewFeedHandler(feedService)
	feedGroup := r.Group("/feed")
	feedGroup.Use(jwt.SoftJWTAuth(accountRepository, cache))

	feedGroup.POST("/listLatest", feedHandler.ListLatest)
	feedGroup.POST("/listLikesCount", feedHandler.LekiListCount)
	feedGroup.POST("/listByPopularity", feedHandler.ListByPopularity)
	feedGroup.POST("/listByTag", feedHandler.ListByTag)

	protectedFeedGroup := feedGroup.Group("")
	protectedFeedGroup.Use(jwt.JWTAuth(accountRepository, cache))

	protectedFeedGroup.POST("/listByFollowing", feedHandler.ListByFollowing)

	messageRepo := message.NewRepository(db)
	messageService := message.NewService(messageRepo)
	messageHandler := message.NewHandler(messageService)
	messageGroup := r.Group("/message")
	protectedMessageGroup := messageGroup.Group("")
	protectedMessageGroup.Use(jwt.JWTAuth(accountRepository, cache))
	protectedMessageGroup.POST("/send", messageHandler.Send)
	protectedMessageGroup.POST("/list", messageHandler.List)
	return r

}
