package main

import (
	"os"

	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/messaging-service/app"
	"github.com/zerodayz7/platform/services/messaging-service/config"
	"github.com/zerodayz7/platform/services/messaging-service/internal/di"
	"github.com/zerodayz7/platform/services/messaging-service/internal/router"
	"github.com/zerodayz7/platform/services/messaging-service/internal/websocket"
)

func main() {
	bootLog := shared.InitBootstrapLogger(os.Getenv("ENV"), false)
	defer func() { _ = bootLog.Sync() }()

	if err := config.LoadConfigGlobal(); err != nil {
		bootLog.Fatal("Config load failed", "error", err)
	}

	log := shared.InitLogger(config.AppConfig.Server.Env, false)

	db, closeDB := config.MustInitDB(config.AppConfig.Database)

	wsHub := websocket.NewHub()
	go wsHub.Run()

	container := di.NewContainer(db, log, &config.AppConfig, wsHub)

	messagingApp := app.NewApp(container)

	router.SetupMessagingRoutes(messagingApp, container)

	server.Run(
		messagingApp,
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
