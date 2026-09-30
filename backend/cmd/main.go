package main

import (
	"context"
	"feedsystem_video_go/internal/config"
	"feedsystem_video_go/internal/db"
	apphttp "feedsystem_video_go/internal/http"
	"feedsystem_video_go/internal/middleware/logger"
	rabbitmq "feedsystem_video_go/internal/middleware/rabbitmq"
	rediscache "feedsystem_video_go/internal/middleware/redis"
	"feedsystem_video_go/internal/observability"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	// 初始化 zap 日志（全项目日志统一走 internal/middleware/logger）
	logger.Init()
	// 加载 .env（本地开发）依赖管理
	if err := godotenv.Load(); err != nil {
		logger.Println(".env not found; continuing")
	}

	// 加载配置
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/config.yaml"
	}
	logger.Printf("Loading config from %s", configPath)
	cfg, usedDefault, err := config.LoadLocalDev(configPath)
	if err != nil {
		zap.L().Fatal("Failed to load config", zap.Error(err))
	}
	if usedDefault {
		logger.Printf("Config File %s not found, using default local config", configPath)
	} else {
		logger.Printf("Config loaded from file: %s", configPath)
	}

	// 连接数据库
	//logger.Printf("Database config: %v", cfg.Database)
	sqlDB, err := db.NewDB(cfg.Database)
	if err != nil {
		zap.L().Fatal("Failed to connect database", zap.Error(err))
	}
	if err := db.AutoMigrate(sqlDB); err != nil {
		zap.L().Fatal("Failed to auto migrate database", zap.Error(err))
	}
	defer db.CloseDB(sqlDB)

	// 连接 Redis (可选，用于缓存)
	cache, err := rediscache.NewFromEnv(&cfg.Redis)
	if err != nil {
		zap.L().Error("Redis config error (cache disabled)", zap.Error(err))
		cache = nil
	} else {
		pingCtx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()
		if err := cache.Ping(pingCtx); err != nil {
			zap.L().Error("Redis not available (cache disabled)", zap.Error(err))
			_ = cache.Close()
			cache = nil
		} else {
			defer cache.Close()
			logger.Printf("Redis connected (cache enabled)")
		}
	}

	// 连接 RabbitMQ (可选，用于消息队列)
	rmq, err := rabbitmq.NewRabbitMQ(&cfg.RabbitMQ)
	if err != nil {
		zap.L().Error("RabbitMQ config error (disabled)", zap.Error(err))
		rmq = nil
	} else {
		defer rmq.Close()
		logger.Printf("RabbitMQ connected")
	}
	// Pprof
	pprofServer, err := observability.NewPprofServer(
		"API",
		cfg.ObservabilityConfig.Pprof.Enabled,
		cfg.ObservabilityConfig.Pprof.ApiAddr,
	)
	if err != nil {
		zap.L().Error("Failed to start API pprof server", zap.Error(err))
	}
	if pprofServer != nil {
		defer pprofServer.Close()
	}

	// 设置路由
	r := apphttp.SetRouter(sqlDB, cache, rmq)
	logger.Printf("Server is running on port %d", cfg.Server.Port)
	if err := r.Run(":" + strconv.Itoa(cfg.Server.Port)); err != nil {
		zap.L().Fatal("Failed to run server", zap.Error(err))
	}
}
