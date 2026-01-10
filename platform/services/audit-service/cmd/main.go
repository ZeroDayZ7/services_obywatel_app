package main

import (
	"os"

	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/utils"
	"github.com/zerodayz7/platform/services/audit-service/config"
	"github.com/zerodayz7/platform/services/audit-service/internal/di"
	"github.com/zerodayz7/platform/services/audit-service/internal/router"
)

func main() {
	// 0. Boostrap Logger
	bootLog := shared.InitBootstrapLogger(os.Getenv("ENV"))
	defer func() { _ = bootLog.Sync() }()
	// 1. Config
	if err := config.LoadConfigGlobal(); err != nil {
		bootLog.Fatal("Config load failed", "error", err)
	}

	// Inicjalizacja loggera
	log := shared.InitLogger(config.AppConfig.Server.Env)

	// Inicjalizacja bazy danych
	db, closeDB := config.MustInitDB(config.AppConfig.Database)
	defer closeDB()

	// Inicjalizacja kontenera DI
	container := di.NewContainer(db, nil, log, &config.AppConfig)

	// Bezpieczne uruchomienie workera w tle
	utils.SafeGo(log, container.AuditWorker.Start)

	// Inicjalizacja aplikacji Fiber i routing
	app := config.NewAuditApp(container)
	router.SetupRoutes(app, container)

	// Konfiguracja Graceful Shutdown
	server.SetupGracefulShutdown(app, config.AppConfig.Shutdown, closeDB)

	// Uruchomienie serwera
	address := ":" + config.AppConfig.Server.Port
	log.Info("Service started", map[string]any{
		"app":     config.AppConfig.Server.AppName,
		"address": address,
		"env":     config.AppConfig.Server.Env,
	})

	if err := app.Listen(address); err != nil {
		log.ErrorObj("Critical server failure", err)
	}
}
