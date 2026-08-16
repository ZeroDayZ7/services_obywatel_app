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

	// 1. Inicjalizacja konfiguracji
	if err := config.LoadConfigGlobal(); err != nil {
		log.Error("❌ Nie udało się załadować konfiguracji", "error", err)
		os.Exit(1)
	}

	// Inicjalizacja magazynu kluczy ze wspólnego pakietu httpserver
	keyStore := httpserver.NewKeyStore()

	// =========================================================================
	// 2. KMS SETUP & FETCH ALL REQUIRED HMAC KEYS
	// =========================================================================
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	kmsCfg := config.AppConfig.ToKMSServiceConfig()

	log.Info("🔍 Sprawdzanie stanu serwisu KMS...")
	if err := kms.HealthCheck(ctx, kmsCfg); err != nil {
		log.Error("❌ KMS Health Check nie powiódł się", "error", err)
		os.Exit(1)
	}

	// 2a. Pobieranie kluczy HMAC w pętli dla wszystkich wymaganych relacji
	for serviceID, targetKey := range config.AppConfig.HMAC.TargetKeys {
		log.Info("🔑 Pobieranie klucza HMAC z KMS...", "service", serviceID, "target_key", targetKey)

		hmacKey, version, err := kms.FetchSymmetricKeyWithVersion(ctx, kmsCfg, targetKey, "HmacSha256")
		if err != nil {
			log.Error("❌ Nie udało się pobrać klucza HMAC z KMS", "service", serviceID, "target_key", targetKey, "error", err)
			os.Exit(1)
		}

		keyStore.SetKey(serviceID, hmacKey, version)
		log.Info("✅ Klucz HMAC załadowany", "service", serviceID, "version", version)
	}

	// 3. Budowanie kontenera DI
	container := di.BuildContainer(&config.AppConfig, keyStore)

	// 4. Budowanie routera HTTP
	r := router.NewRouter(container)

	// 5. Konfiguracja serwera HTTP
	srv := &http.Server{
		Addr:         ":" + config.AppConfig.Server.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 6. Graceful Shutdown
	if err := httpserver.Run(srv, config.AppConfig.Shutdown); err != nil {
		log.Error("Server forced shutdown with error", "error", err)
	}
}
