package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/kms"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/services/officer-bff/config"
	"github.com/zerodayz7/services/officer-bff/internal/di"
	"github.com/zerodayz7/services/officer-bff/internal/router"
)

func main() {
	log := shared.GetLogger()

	// 1. Inicjalizacja konfiguracji (bez bazy danych)
	app := config.InitApp()

	// =========================================================================
	// 2. KMS SETUP & FETCH HMAC KEY
	// =========================================================================
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	kmsCfg := app.Config.ToKMSServiceConfig()

	log.Info("🔍 Sprawdzanie stanu serwisu KMS...")
	if err := kms.HealthCheck(ctx, kmsCfg); err != nil {
		log.Error("❌ KMS Health Check nie powiódł się", "error", err)
		os.Exit(1)
	}

	log.Info("🔑 Pobieranie klucza HMAC z KMS dla officer-bff...")
	hmacKey, version, err := kms.FetchSymmetricKeyWithVersion(ctx, kmsCfg, "hmac-gateway-officer-bff", "HmacSha256")
	if err != nil {
		log.Error("❌ Nie udało się pobrać klucza HMAC z KMS", "error", err)
		os.Exit(1)
	}

	app.Config.Internal.HMACSecret = string(hmacKey)
	log.Info("✅ Klucz HMAC pobrany pomyślnie z KMS", "version", version)

	// 3. Budowanie kontenera DI
	container := di.BuildContainer(app)

	// 4. Budowanie routera HTTP
	r := router.NewRouter(container)

	// 5. Konfiguracja serwera HTTP
	srv := &http.Server{
		Addr:         ":" + app.Config.Server.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 6. Graceful Shutdown
	if err := httpserver.Run(srv, app.Config.Shutdown); err != nil {
		log.Error("Server forced shutdown with error", "error", err)
	}
}
