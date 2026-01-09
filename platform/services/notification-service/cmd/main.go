package main

import (
	"os"

	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/notification-service/config"
	"github.com/zerodayz7/platform/services/notification-service/internal/di"
	"github.com/zerodayz7/platform/services/notification-service/internal/router"
)

func main() {

	// Logger
	log := shared.InitLogger(os.Getenv("ENV"))

	// Load Config
	if err := config.LoadConfigGlobal(); err != nil {
		log.ErrorObj("Config load failed", err)
		return
	}

	// Redis
	redisClient, err := redis.New(redis.Config(config.AppConfig.Redis))
	if err != nil {
		log.ErrorObj("Redis init failed", err)
		return
	}
	defer redisClient.Close()

	// Database
	db, closeDB := config.MustInitDB(config.AppConfig.Database)
	defer closeDB()

	// Dependency Injection
	container := di.NewContainer(db, redisClient, log)

	// Start Notification Worker w tle
	go container.Workers.NotificationWorker.Start()

	// Fiber App
	app := config.NewNotificationApp(config.AppConfig.Server)

	// Routes
	router.SetupRoutes(app, container.Handlers.NotificationHandler)

	// Graceful Shutdown
	server.SetupGracefulShutdown(app, closeDB, config.AppConfig.Shutdown)
	// Start Server

	address := "0.0.0.0:" + config.AppConfig.Server.Port
	log.InfoObj("Server started", map[string]any{
		"app":     config.AppConfig.Server.AppName,
		"address": address,
	})
	if err := app.Listen(address); err != nil {
		log.ErrorObj("Failed to start server", err)
	}
}
