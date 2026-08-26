package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/messaging-service/app"
	"github.com/zerodayz7/platform/services/messaging-service/config"
	"github.com/zerodayz7/platform/services/messaging-service/internal/di"
	"github.com/zerodayz7/platform/services/messaging-service/internal/router"
	"github.com/zerodayz7/platform/services/messaging-service/internal/websocket"
)

func main() {
	// 0. Bootstrap Logger
	bootLog := shared.InitBootstrapLogger(os.Getenv("ENV"), false)
	defer func() { _ = bootLog.Sync() }()

	// 1. Load Config
	if err := config.LoadConfigGlobal(); err != nil {
		bootLog.Fatal("Config load failed", "error", err)
	}

	log := shared.GetLogger()

	// 2. Init KeyStore & Signal Context
	keyStore := httpserver.NewKeyStore()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	securityCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 3. Ładowanie Kluczy z KMS do KeyStore
	LoadSecurityKeys(securityCtx, &config.AppConfig, keyStore)

	// 4. Database
	db, closeDB := config.MustInitDB(config.AppConfig.Database)

	// 5. WebSocket Hub
	wsHub := websocket.NewHub()
	go wsHub.Run()

	// 6. DI Container & App Setup (zgodnie z obecną sygnaturą NewContainer - 4 argumenty)
	container := di.NewContainer(db, log, &config.AppConfig, wsHub)
	messagingApp := app.NewApp(container)

	router.SetupMessagingRoutes(messagingApp, container)

	// 7. Run Server
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
