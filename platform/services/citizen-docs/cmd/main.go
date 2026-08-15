package main

import (
	"context"
	"os"
	"time"

	"github.com/zerodayz7/platform/pkg/envelope"
	"github.com/zerodayz7/platform/pkg/kms"
	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/citizen-docs/app"
	"github.com/zerodayz7/platform/services/citizen-docs/config"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/di"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/router"
)

func main() {
	// 0. Bootstrap Logger
	bootLog := shared.InitBootstrapLogger(os.Getenv("ENV"), false)
	defer func() { _ = bootLog.Sync() }()

	// 1. Load Config
	if err := config.LoadConfigGlobal(); err != nil {
		bootLog.Fatal("Config load failed", "error", err)
	}

	log := shared.InitLogger(config.AppConfig.Server.Env, false)

	// =========================================================================
	// 2. KMS SETUP & HEALTH CHECK
	// =========================================================================
	kmsCfg := kms.Config{
		Endpoint:      config.AppConfig.KMS.Endpoint,
		ServiceName:   config.AppConfig.Server.AppName, // "citizen-docs-service"
		ServiceSecret: config.AppConfig.KMS.ServiceSecret,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	bootLog.Info("🔍 Sprawdzanie stanu serwisu KMS...")
	if err := kms.HealthCheck(ctx, kmsCfg); err != nil {
		bootLog.Fatal("❌ KMS Health Check nie powiódł się. Przerywanie startu serwisu", "error", err)
	}
	bootLog.Info("✅ KMS jest dostępny i gotowy do pracy")

	// =========================================================================
	// 3. FETCH INTERNAL HMAC SECRET Z KMS
	// =========================================================================
	bootLog.Info("🔑 Pobieranie klucza 'hmac-gateway-docs' z KMS...")

	// DODANO czwarty parametr: "HmacSha256" (zamiast domyślnego "AES256GCM")
	internalHMACKey, err := kms.FetchSymmetricKey(ctx, kmsCfg, "hmac-gateway-docs", "HmacSha256")
	if err != nil {
		bootLog.Fatal("❌ Nie udało się pobrać klucza HMAC z KMS", "error", err)
	}

	config.AppConfig.Internal.HMACSecret = string(internalHMACKey)
	bootLog.Info("✅ Klucz HMAC komunikacji wewnętrznej pobrany pomyślnie")

	// =========================================================================
	// 4. INICJALIZACJA CRYPTOR, BAZY DANYCH I DI
	// =========================================================================
	cryptor := envelope.NewEnvelopeCryptor(kmsCfg)

	db, closeDB := config.MustInitDB(config.AppConfig.Database)
	defer closeDB()

	container := di.NewContainer(db, log, &config.AppConfig, cryptor)

	docsApp := app.NewDocsApp(container)
	router.SetupDocsRoutes(docsApp, container)

	// =========================================================================
	// 5. URUCHOMIENIE SERWERA
	// =========================================================================
	server.Run(
		docsApp,
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
