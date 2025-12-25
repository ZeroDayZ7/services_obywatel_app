package main

import (
	"os"

	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/notification-service/config"
	"github.com/zerodayz7/platform/services/notification-service/internal/di"
	"github.com/zerodayz7/platform/services/notification-service/internal/router"
)

func main() {
	// Inicjalizacja loggera
	log := shared.InitLogger(os.Getenv("ENV"))

	// Config
	if err := config.LoadConfigGlobal(); err != nil {
		log.ErrorObj("Config load failed", err)
		return
	}

	// DB
	db, closeDB := config.MustInitDB()
	defer closeDB()

	// Dependency Injection
	container := di.NewContainer(db)

	// Fiber
	app := config.NewNotificationApp()

	// Routes
	router.SetupRoutes(app, container.NotificationHandler)

	// Graceful shutdown
	server.SetupGracefulShutdown(app, closeDB, config.AppConfig.Shutdown)

	address := "0.0.0.0:" + config.AppConfig.Server.Port
	log.InfoObj("notification-Server listening", map[string]any{"address": address})

	// Start serwera
	if err := app.Listen(address); err != nil {
		log.ErrorObj("Failed to start server", err)
	}
}
