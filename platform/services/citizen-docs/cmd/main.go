package main

import (
	"os"

	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/citizen-docs/config"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/di"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/router"
)

func main() {
	bootLog := shared.InitBootstrapLogger(os.Getenv("ENV"))
	defer func() { _ = bootLog.Sync() }()

	if err := config.LoadConfigGlobal(); err != nil {
		bootLog.Fatal("Config load failed", "error", err)
	}

	log := shared.InitLogger(config.AppConfig.Server.Env)

	db, closeDB := config.MustInitDB(config.AppConfig.Database)
	defer closeDB()

	container := di.NewContainer(db, log, &config.AppConfig)

	app := config.NewDocsApp(container)

	router.SetupDocsRoutes(app, container.UserDocumentSvc)

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
		},
	)
}
