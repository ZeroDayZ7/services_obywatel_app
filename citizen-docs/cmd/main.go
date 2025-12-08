package main

import (
	"os"

	"github.com/zerodayz7/http-server/config"
	"github.com/zerodayz7/http-server/internal/di"
	"github.com/zerodayz7/http-server/internal/router"
	"github.com/zerodayz7/http-server/internal/server"
	"github.com/zerodayz7/http-server/internal/shared/logger"
	"go.uber.org/zap"
)

func main() {
	// Init logger
	_, _ = logger.InitLogger(os.Getenv("ENV"))

	// Load global config
	if err := config.LoadConfigGlobal(); err != nil {
		logger.GetLogger().Fatal("Config load failed", zap.Error(err))
	}

	// Initialize DB
	db, closeDB := config.MustInitDB()
	defer closeDB()

	// Dependency Injection container
	container := di.NewContainer(db)

	// Fiber app
	app := config.NewFiberApp()

	// Routes
	router.SetupDocsRoutes(app, container.UserDocumentRepo)

	// Graceful shutdown
	server.SetupGracefulShutdown(app, closeDB, config.AppConfig.Shutdown)

	address := "0.0.0.0:" + config.AppConfig.Server.Port
	logger.GetLogger().Info(
		"Citizen-Docs Microservice listening",
		zap.String("address", address),
	)
	if err := app.Listen(address); err != nil {
		logger.GetLogger().Fatal("Failed to start server", zap.Error(err))
	}
}
