package main

import (
	"fmt"

	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/utils"
	"github.com/zerodayz7/platform/services/notification-service/config"
	"github.com/zerodayz7/platform/services/notification-service/internal/di"
	"github.com/zerodayz7/platform/services/notification-service/internal/router"
)

func main() {
	// 1. Config
	if err := config.LoadConfigGlobal(); err != nil {
		panic(fmt.Sprintf("Config load failed: %v", err))
	}

	// 2. Logger
	log := shared.InitLogger(config.AppConfig.Server.Env)

	// 3. Telemetry (Tracer)
	// if config.AppConfig.OTEL.Enabled {
	// 	cleanup := telemetry.InitTracer(
	// 		config.AppConfig.Server.AppName,
	// 		config.AppConfig.OTEL.Endpoint,
	// 	)
	// 	defer cleanup()
	// }

	// 4. Infrastruktura (Redis & DB)
	redisClient, err := redis.New(redis.Config(config.AppConfig.Redis))
	if err != nil {
		log.ErrorObj("Redis failed", err)
	}
	defer redisClient.Close()

	db, closeDB := config.MustInitDB(config.AppConfig.Database)
	defer closeDB()

	// Dependency Injection
	container := di.NewContainer(db, redisClient, log, &config.AppConfig)

	// Start Notification Worker w tle
	utils.SafeGo(log, container.Workers.NotificationWorker.Start)

	// Fiber App
	app := config.NewNotificationApp(container)

	// Routes
	router.SetupRoutes(app, container)

	// Graceful Shutdown
	server.SetupGracefulShutdown(
		app,
		config.AppConfig.Shutdown,
		closeDB,
		func() { _ = redisClient.Close() },
	)
	// Start Server

	address := "0.0.0.0:" + config.AppConfig.Server.Port
	log.Info("Service started", map[string]any{
		"app":     config.AppConfig.Server.AppName,
		"version": config.AppConfig.Server.AppVersion,
		"address": address,
		"env":     config.AppConfig.Server.Env,
	})
	if err := app.Listen(address); err != nil {
		log.ErrorObj("Failed to start server", err)
	}
}
