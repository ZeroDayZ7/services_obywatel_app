package main

import (
	"os"

	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/gateway/config"
	"github.com/zerodayz7/platform/services/gateway/internal/router"
	"github.com/zerodayz7/platform/services/gateway/internal/server"
)

func main() {
	// Inicjalizacja loggera
	log := shared.InitLogger(os.Getenv("ENV"))

	// Config
	if err := config.LoadConfigGlobal(); err != nil {
		log.ErrorObj("Config load failed", err)
		return
	}

	// Fiber app
	app := config.NewFiberApp()

	// Routes
	router.SetupRoutes(app)

	// Graceful shutdown
	server.SetupGracefulShutdown(app, nil, config.AppConfig.Shutdown)

	// Log start
	address := "0.0.0.0:" + config.AppConfig.Server.Port
	log.InfoObj("Gateway-Server listening", map[string]any{"address": address})

	// Start serwera
	if err := app.Listen(address); err != nil {
		log.ErrorObj("Failed to start server", err)
	}
}
