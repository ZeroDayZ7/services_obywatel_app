package main

import (
	"os"

	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/telemetry"
	"github.com/zerodayz7/platform/services/auth-service/config"
	"github.com/zerodayz7/platform/services/auth-service/internal/di"
	"github.com/zerodayz7/platform/services/auth-service/internal/router"
)

func main() {
	// Inicjalizacja loggera
	log := shared.InitLogger(os.Getenv("ENV"))

	// Config
	if err := config.LoadConfigGlobal(); err != nil {
		log.Fatal("Config load failed", err)
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

	// DB
	db, closeDB := config.MustInitDB(config.AppConfig.Database)
	defer closeDB()

	// Dependency Injection

	container := di.NewContainer(db, redisClient, &config.AppConfig)
	// Fiber
	app := config.NewAuthApp(container)

	// Routes
	router.SetupRoutes(app, container)
	// Graceful shutdown
	server.SetupGracefulShutdown(app, closeDB, config.AppConfig.Shutdown)

	address := "0.0.0.0:" + config.AppConfig.Server.Port
	log.Info("Server started", map[string]any{
		"app":     config.AppConfig.Server.AppName,
		"address": address,
	})
	// Start serwera
	if err := app.Listen(address); err != nil {
		log.ErrorObj("Failed to start server", err)
	}
}
