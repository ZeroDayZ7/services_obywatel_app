package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/kms"
	"github.com/zerodayz7/platform/pkg/rabbitmq"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/services/identity-service/config"
	"github.com/zerodayz7/services/identity-service/internal/di"
	"github.com/zerodayz7/services/identity-service/internal/router"
)

func main() {
	log := shared.GetLogger()

	// 1. Inicjalizacja konfiguracji i bazy danych
	app, closeDB := config.InitApp()
	defer closeDB()

	// =========================================================================
	// 2. KMS SETUP & FETCH KEYS
	// =========================================================================
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	kmsCfg := app.Config.ToKMSServiceConfig()

	log.Info("🔍 Sprawdzanie stanu serwisu KMS...")
	if err := kms.HealthCheck(ctx, kmsCfg); err != nil {
		log.Error("❌ KMS Health Check nie powiódł się", "error", err)
		os.Exit(1)
	}

	// 2a. Pobranie klucza HMAC do komunikacji z Gatewayem
	gatewayHmacKey, version, err := kms.FetchSymmetricKeyWithVersion(ctx, kmsCfg, "hmac-gateway-identity", "HmacSha256")
	if err != nil {
		log.Error("❌ Nie udało się pobrać klucza HMAC Gateway z KMS", "error", err)
		os.Exit(1)
	}
	app.Config.Internal.HMACSecret = string(gatewayHmacKey)
	log.Info("✅ Klucz HMAC Gateway pobrany pomyślnie z KMS", "version", version)

	// 2b. Pobranie klucza HMAC do Blind Indexu PESEL
	peselHmacKey, peselKeyVersion, err := kms.FetchSymmetricKeyWithVersion(ctx, kmsCfg, "identity-pesel-blind-index", "HmacSha256")
	if err != nil {
		log.Error("❌ Nie udało się pobrać klucza HMAC dla PESEL z KMS", "error", err)
		os.Exit(1)
	}
	log.Info("✅ Klucz HMAC dla PESEL pobrany pomyślnie z KMS", "version", peselKeyVersion)

	// 2c. Pobranie klucza HMAC dla RabbitMQ Publishera
	rabbitHMACKey, rabbitKeyVersion, err := kms.FetchSymmetricKeyWithVersion(ctx, kmsCfg, "hmac-identity-rabbitmq", "HmacSha256")
	if err != nil {
		log.Error("❌ Nie udało się pobrać klucza HMAC RabbitMQ z KMS", "error", err)
		os.Exit(1)
	}
	log.Info("✅ Klucz HMAC dla RabbitMQ pobrany pomyślnie z KMS", "version", rabbitKeyVersion)

	// =========================================================================
	// 3. RABBITMQ SETUP
	// =========================================================================
	var eventPublisher rabbitmq.EventPublisher
	if app.Config.RabbitMQ.Enabled {
		log.Info("RabbitMQ is ENABLED. Connecting...")
		eventPublisher, err = rabbitmq.NewLivePublisher(
			app.Config.RabbitMQ.GetURL(),
			app.Config.Server.AppName, // np. "identity-service"
			rabbitHMACKey,
		)
		if err != nil {
			log.Error("❌ Nie udało się połączyć z RabbitMQ", "error", err)
			os.Exit(1)
		}
	} else {
		log.Warn("RabbitMQ is DISABLED. Fallback to No-Op Driver.")
		eventPublisher = rabbitmq.NewNoOpPublisher()
	}
	defer func() {
		if err := eventPublisher.Close(); err != nil {
			log.Error("Błąd podczas zamykania połączenia z RabbitMQ", "error", err)
		}
	}()

	// =========================================================================
	// 4. DI CONTAINER & ROUTER
	// =========================================================================
	container := di.BuildContainer(app, eventPublisher, peselHmacKey, kmsCfg)

	// 5. Budowanie routera na natywnym http.ServeMux
	r := router.NewRouter(container)

	// 6. Konfiguracja instancji serwera HTTP
	srv := &http.Server{
		Addr:         ":" + app.Config.Server.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 7. Uruchomienie z Graceful Shutdown przez pkg/httpserver
	if err := httpserver.Run(srv, app.Config.Shutdown); err != nil {
		log.Error("Server forced shutdown with error", "error", err)
	}
}
