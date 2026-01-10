package main

import (
	"os"

	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/telemetry"
	"github.com/zerodayz7/platform/services/gateway/config"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
	"github.com/zerodayz7/platform/services/gateway/internal/router"
)

func main() {
	log := shared.InitLogger(os.Getenv("ENV"))

	// Config
	if err := config.LoadConfigGlobal(); err != nil {
		log.ErrorObj("Config load failed", err)
		return
	}

	// OTP
	cleanup := telemetry.InitTracer(
		config.AppConfig.Server.AppName,
		config.AppConfig.OTEL.Endpoint,
	)
	defer cleanup()

	// Redis – z nowego pkg
	redisClient, err := redis.New(redis.Config(config.AppConfig.Redis))
	if err != nil {
		log.ErrorObj("Redis failed", err)
	}
	defer redisClient.Close()

	container := di.NewContainer(redisClient, &config.AppConfig)

	app := config.NewGatewayApp(container)

	// Routes
	router.SetupRoutes(app, container)

	// Graceful shutdown
	server.SetupGracefulShutdown(app, nil, config.AppConfig.Shutdown)

	// Log start
	address := "0.0.0.0:" + config.AppConfig.Server.Port
	log.Info("Server started", map[string]any{
		"app":     config.AppConfig.Server.AppName,
		"version": config.AppConfig.Server.AppVersion,
		"address": address,
	})
	// Start serwera
	if err := app.Listen(address); err != nil {
		log.ErrorObj("Failed to start server", err)
	}
}
