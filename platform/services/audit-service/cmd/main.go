package main

import (
	"context"
	"os"

	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/audit-service/config"
	"github.com/zerodayz7/platform/services/audit-service/internal/di"
	"github.com/zerodayz7/platform/services/audit-service/internal/router"
)

func main() {
	// Inicjalizacja loggera
	log := shared.InitLogger(os.Getenv("ENV"))

	// Config
	if err := config.LoadConfigGlobal(); err != nil {
		log.ErrorObj("Config load failed", err)
		return
	}

	// Redis – z nowego pkg
	redisClient, err := redis.New(redis.Config(config.AppConfig.Redis))
	if err != nil {
		log.ErrorObj("Redis failed", err)
	}
	defer redisClient.Close()

	// DB
	db, closeDB := config.MustInitDB(config.AppConfig.Database)
	defer closeDB()

	// 🔧 TU: Tworzymy tabelę jeśli jej brak
	ctx := context.Background()
	if err := config.EnsureAuditTable(ctx, db); err != nil {
		log.ErrorObj("Failed to ensure audit table exists", err)
		return
	}

	// Dependency Injection
	container := di.NewContainer(db, redisClient, log)

	// START WORKERA (w tle)
	go container.AuditWorker.Start()

	// Fiber
	app := config.NewAuditApp(config.AppConfig.Server)

	// 1. Health
	router.SetupHealthRoutes(app)

	// 2. Audit - trasy biznesowe
	router.SetupAuditRoutes(app, container.AuditHandler)

	// 3. Fallback
	router.SetupFallbackHandlers(app)

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
