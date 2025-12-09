package main

import (
	"os"

	"github.com/zerodayz7/platform/pkg/shared"

	"github.com/zerodayz7/platform/services/auth-service/config"
	"github.com/zerodayz7/platform/services/auth-service/internal/di"
	"github.com/zerodayz7/platform/services/auth-service/internal/router"
	"github.com/zerodayz7/platform/services/auth-service/internal/server"
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
	app := config.NewFiberApp()

	// Routes
	router.SetupRoutes(app, container.AuthHandler, container.UserHandler)

	// Graceful shutdown
	server.SetupGracefulShutdown(app, closeDB, config.AppConfig.Shutdown)

	address := "0.0.0.0:" + config.AppConfig.Server.Port
	log.InfoObj("Auth-Server listening", map[string]any{"address": address})

	// Start serwera
	if err := app.Listen(address); err != nil {
		log.ErrorObj("Failed to start server", err)
	}
}
