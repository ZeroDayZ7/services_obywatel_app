package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/v3/server"
	"github.com/zerodayz7/platform/services/websocket-service/app"
	"github.com/zerodayz7/platform/services/websocket-service/internal/config"
	"github.com/zerodayz7/platform/services/websocket-service/internal/hub"
	"github.com/zerodayz7/platform/services/websocket-service/internal/router"
)

func main() {
	// 0. Bootstrap logger setup
	bootLog := shared.InitBootstrapLogger(os.Getenv("APP_ENV"), false)
	defer func() { _ = bootLog.Sync() }()

	// 1. Load configuration
	if err := config.LoadConfigGlobal(); err != nil {
		bootLog.Error("Config load failed", "error", err)
		os.Exit(1)
	}

	log := shared.InitLogger(config.AppConfig.Server.Env, false)

	// 2. Redis connection for Pub/Sub
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", config.AppConfig.Redis.Host, config.AppConfig.Redis.Port),
		Username: config.AppConfig.Redis.Username,
		Password: config.AppConfig.Redis.Password,
		DB:       config.AppConfig.Redis.DB,
	})
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Error("Failed to close Redis client", "error", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Warn("Redis is not available for websocket-service; operating in memory-only mode", "error", err)
	} else {
		log.Info("Redis Pub/Sub connected for websocket-service")
	}

	// 3. Hub & Subscriber background routine
	hubManager := hub.NewHub(redisClient)
	go hubManager.RunRedisSubscriber(context.Background())

	// 4. Fiber App & Routes initialization
	wsApp := app.NewApp(&config.AppConfig)
	router.SetupRoutes(wsApp, hubManager, &config.AppConfig)

	// 5. Start server using shared package/server v3 framework
	server.Run(
		wsApp,
		server.Config{
			Port:       config.AppConfig.Server.Port,
			AppName:    config.AppConfig.Server.AppName,
			AppVersion: config.AppConfig.Server.AppVersion,
			Env:        config.AppConfig.Server.Env,
			Shutdown:   config.AppConfig.Shutdown,
		},
		*log,
		nil,
	)
}
