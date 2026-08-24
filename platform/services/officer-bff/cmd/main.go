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

	// Pobieranie kluczy HMAC w pętli dla wszystkich wymaganych relacji
	for serviceID, target := range config.AppConfig.HMAC.TargetKeys {
		hmacKey, version, err := kms.FetchSymmetricKeyWithVersion(ctx, kmsCfg, target.TargetKey, 1, target.Algorithm)
		if err != nil {
			log.Error("❌ Nie udało się pobrać klucza HMAC z KMS", "service", serviceID, "target_key", target.TargetKey, "error", err)
			os.Exit(1)
		}

		keyStore.SetKey(serviceID, hmacKey, uint32(version))
		log.Info("✅ Klucz HMAC załadowany", "service", serviceID, "version", version)
	}

	// 3. Budowanie kontenera DI
	container := di.BuildContainer(&config.AppConfig, keyStore)

	// 4. Budowanie routera HTTP
	r := router.NewRouter(container)

	// 5. Konfiguracja serwera HTTP z wykorzystaniem wartości z pliku konfiguracji
	srv := &http.Server{
		Addr:         ":" + config.AppConfig.Server.Port,
		Handler:      r,
		ReadTimeout:  config.AppConfig.Server.ReadTimeout,
		WriteTimeout: config.AppConfig.Server.WriteTimeout,
		IdleTimeout:  config.AppConfig.Server.IdleTimeout,
	}

	// 6. Graceful Shutdown
	if err := httpserver.Run(srv, config.AppConfig.Shutdown); err != nil {
		log.Error("Server forced shutdown with error", "error", err)
	}
}
