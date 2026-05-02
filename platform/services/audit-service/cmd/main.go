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
	// Bootstrap logger for startup errors
	bootLog := shared.InitBootstrapLogger(os.Getenv("ENV"), false)
	defer func() { _ = bootLog.Sync() }()

	// Load global configuration
	if err := config.LoadConfigGlobal(); err != nil {
		bootLog.Fatal("Config load failed", "error", err)
	}

	// Initialize production logger
	log := shared.InitLogger(config.AppConfig.Server.Env, false)

	// Initialize Database
	db, closeDB := config.MustInitDB(config.AppConfig.Database)
	defer closeDB()

	// Initialize DI container
	container := di.NewContainer(db, nil, log, &config.AppConfig)

	// Start background worker
	utils.SafeGo(log, container.AuditWorker.Start)

	// Initialize app and routes
	app := config.NewAuditApp(container)
	router.SetupRoutes(app, container)

	// Start server with unified run handler
	server.Run(
		app,
		server.Config{
			Port:       config.AppConfig.Server.Port,
			AppName:    config.AppConfig.Server.AppName,
			AppVersion: config.AppConfig.Server.AppVersion,
			Env:        config.AppConfig.Server.Env,
			Shutdown:   config.AppConfig.Shutdown,
		},
		*log,
		func() {
			closeDB()
			// Additional resource cleanup can be added here
		},
	)
}
