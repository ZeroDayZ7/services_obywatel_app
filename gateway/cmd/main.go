package main

import (
	"os"

	"github.com/zerodayz7/http-server/config"
	"github.com/zerodayz7/http-server/internal/router"
	"github.com/zerodayz7/http-server/internal/server"
	"github.com/zerodayz7/http-server/internal/shared/logger"
	"go.uber.org/zap"
)

func main() {
	_, _ = logger.InitLogger(os.Getenv("ENV"))

	// Config
	if err := config.LoadConfigGlobal(); err != nil {
		logger.GetLogger().Fatal("Config load failed", zap.Error(err))
	}

	// Fiber
	app := config.NewFiberApp()

	// Routes
	router.SetupRoutes(app)

	// Graceful shutdown
	server.SetupGracefulShutdown(app, config.AppConfig.Shutdown)

	address := "0.0.0.0:" + config.AppConfig.Server.Port
	logger.GetLogger().Info(
		"Gateway-Server listening",
		zap.String("address", address),
	)
	if err := app.Listen(address); err != nil {
		logger.GetLogger().Fatal("Failed to start server", zap.Error(err))
	}
}
