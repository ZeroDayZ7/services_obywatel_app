package main

import (
	"context"
	"os"
	"time"

	"github.com/zerodayz7/platform/pkg/kms"
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

	// 1. Config
	if err := config.LoadConfigGlobal(); err != nil {
		bootLog.Fatal("Config load failed", "error", err)
	}

	// =========================================================================
	// 2. KMS SETUP & FETCH INTERNAL HMAC KEY
	// =========================================================================
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	kmsCfg := kms.Config{
		Endpoint:      config.AppConfig.KMS.Endpoint,
		ServiceName:   config.AppConfig.Server.AppName, // "messaging-service"
		ServiceSecret: config.AppConfig.KMS.ServiceSecret,
	}

	bootLog.Info("🔍 Sprawdzanie stanu serwisu KMS...")
	if err := kms.HealthCheck(ctx, kmsCfg); err != nil {
		bootLog.Error("❌ KMS Health Check nie powiódł się", "error", err)
		os.Exit(1)
	}

	bootLog.Info("🔑 Pobieranie klucza 'hmac-gateway-messaging' z KMS...")
	internalHMACKey, err := kms.FetchSymmetricKey(ctx, kmsCfg, "hmac-gateway-messaging")
	if err != nil {
		bootLog.Error("❌ Nie udało się pobrać klucza HMAC z KMS", "error", err)
		os.Exit(1)
	}

	config.AppConfig.Internal.HMACSecret = string(internalHMACKey)
	bootLog.Info("✅ Klucz HMAC komunikacji wewnętrznej pobrany pomyślnie")

	// 3. Logger
	log := shared.InitLogger(config.AppConfig.Server.Env, false)

	// 4. Database
	db, closeDB := config.MustInitDB(config.AppConfig.Database)

	// 5. WebSocket Hub
	wsHub := websocket.NewHub()
	go wsHub.Run()

	// 6. DI Container & App Setup
	container := di.NewContainer(db, log, &config.AppConfig, wsHub)
	messagingApp := app.NewApp(container)

	router.SetupMessagingRoutes(messagingApp, container)

	// 7. Run server
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
