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
	"github.com/zerodayz7/platform/services/messaging-service/internal/security"
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

	// 3. Ładowanie Kluczy z KMS do KeyStore (użycie funkcji z internal/security)
	rabbitHMACKey, err := security.LoadSecurityKeys(securityCtx, &config.AppConfig, keyStore)
	if err != nil {
		log.Error("❌ Nie udało się załadować kluczy bezpieczeństwa z KMS", "error", err)
		os.Exit(1)
	}
	if config.AppConfig.RabbitMQEnabled && len(rabbitHMACKey) == 0 {
		log.Error("❌ Brak klucza RabbitMQ po załadowaniu z KMS, mimo że RabbitMQ jest włączone")
		os.Exit(1)
	}
	if !config.AppConfig.RabbitMQEnabled && rabbitHMACKey != nil {
		log.Warn("RabbitMQ jest wyłączony, więc zwrócony klucz HMAC powinien być pusty.")
	}

	// 4. Database
	db, closeDB := config.MustInitDB(config.AppConfig.Database)
	defer closeDB()

	// 5. WebSocket Hub
	wsHub := websocket.NewHub()
	go wsHub.Run()

	// 6. DI Container & App Setup
	container := di.NewContainer(db, log, &config.AppConfig, wsHub, keyStore)
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
		nil,
	)
}
