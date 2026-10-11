package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/websocket-service/app"
	"github.com/zerodayz7/platform/services/websocket-service/internal/config"
	"github.com/zerodayz7/platform/services/websocket-service/internal/hub"
	"github.com/zerodayz7/platform/services/websocket-service/internal/router"
)

func main() {
	bootLog := shared.InitBootstrapLogger(os.Getenv("APP_ENV"), false)
	defer func() { _ = bootLog.Sync() }()

	if err := config.LoadConfigGlobal(); err != nil {
		bootLog.Fatal("Config load failed", "error", err)
	}

	log := shared.GetLogger()
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", config.AppConfig.Redis.Host, config.AppConfig.Redis.Port),
		Username: config.AppConfig.Redis.Username,
		Password: config.AppConfig.Redis.Password,
		DB:       config.AppConfig.Redis.DB,
	})
	defer func() { _ = redisClient.Close() }()

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Warn("Redis is not available for websocket-service; operating in memory-only mode", "error", err)
	} else {
		log.Info("Redis Pub/Sub connected for websocket-service")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	hubManager := hub.NewHub(redisClient)
	go hubManager.RunRedisSubscriber(ctx)

	fiberApp := app.NewApp(&config.AppConfig)
	router.SetupRoutes(fiberApp, hubManager, &config.AppConfig)

	address := fmt.Sprintf("0.0.0.0:%s", config.AppConfig.Server.Port)
	log.Info("Service started", map[string]any{
		"app":     config.AppConfig.Server.AppName,
		"version": config.AppConfig.Server.AppVersion,
		"address": address,
		"env":     config.AppConfig.Server.Env,
	})

	if err := fiberApp.Listen(address); err != nil {
		log.Error("Fiber application failed to start", "error", err)
		os.Exit(1)
	}

	shutdownTimeout := config.AppConfig.Shutdown
	if shutdownTimeout > 0 {
		time.Sleep(shutdownTimeout)
	}
}
