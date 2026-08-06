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
	bootLog := shared.InitBootstrapLogger(os.Getenv("ENV"), false)
	defer func() { _ = bootLog.Sync() }()

	if err := config.LoadConfigGlobal(); err != nil {
		bootLog.Fatal("Config load failed", "error", err)
	}

	log := shared.InitLogger(config.AppConfig.Server.Env, false)

	// =========================================================================
	// 1. KROK: HEALTH CHECK KMS
	// =========================================================================
	kmsCfg := kms.Config{
		Endpoint:      config.AppConfig.KMS.Endpoint,
		ServiceName:   config.AppConfig.Server.AppName, // "citizen-docs-service"
		ServiceSecret: config.AppConfig.KMS.InternalSecret,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	bootLog.Info("🔍 Sprawdzanie stanu serwisu KMS...")
	if err := kms.HealthCheck(ctx, kmsCfg); err != nil {
		bootLog.Fatal("❌ KMS Health Check nie powiódł się. Przerywanie startu serwisu", "error", err)
	}
	bootLog.Info("✅ KMS jest dostępny i gotowy do pracy")

	// =========================================================================
	// 2. KROK: POBRANIE INTERNAL HMAC SECRET Z KMS DO WERYFIKACJI GATEWAYA
	// =========================================================================
	bootLog.Info("🔑 Pobieranie klucza 'internal-communication-hmac' z KMS...")
	internalHMACKey, err := kms.FetchSymmetricKey(ctx, kmsCfg, "internal-communication-hmac")
	if err != nil {
		bootLog.Fatal("❌ Nie udało się pobrać klucza HMAC z KMS", "error", err)
	}

	// Przypisujemy dynamicznie pobrany z KMS klucz do konfiguracji pod middleware weryfikacyjny z Gatewaya
	config.AppConfig.Internal.HMACSecret = string(internalHMACKey)
	bootLog.Info("✅ Klucz HMAC komunikacji wewnętrznej pobrany pomyślnie")

	// =========================================================================
	// 3. INICJALIZACJA ENVELOPE CRYPTOR I BAZY DANYCH
	// =========================================================================
	cryptor := envelope.NewEnvelopeCryptor(kmsCfg)

	db, closeDB := config.MustInitDB(config.AppConfig.Database)
	defer closeDB()

	// Wstrzykujemy zarówno bazę, jak i cryptor do kontenera DI
	container := di.NewContainer(db, log, &config.AppConfig, cryptor)

	docsApp := app.NewDocsApp(container)

	router.SetupDocsRoutes(docsApp, container)

	// =========================================================================
	// 4. URUCHOMIENIE SERWERA
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
		func() {
			closeDB()
		},
	)
}
