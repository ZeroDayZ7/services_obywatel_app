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
	// 2. RABBITMQ SETUP
	// =========================================================================
	rabbitCfg := rabbitmq.Config{
		Enabled:  app.Config.RabbitMQ.Enabled,
		Host:     app.Config.RabbitMQ.Host,
		Port:     app.Config.RabbitMQ.Port,
		User:     app.Config.RabbitMQ.User,
		Password: app.Config.RabbitMQ.Password,
		VHost:    app.Config.RabbitMQ.VHost,
	}

	eventPublisher, err := rabbitmq.NewPublisher(rabbitCfg)
	if err != nil {
		log.Error("❌ Nie udało się połączyć z RabbitMQ", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := eventPublisher.Close(); err != nil {
			log.Error("Błąd podczas zamykania połączenia z RabbitMQ", "error", err)
		}
	}()

	// =========================================================================
	// 3. KMS SETUP & FETCH KEYS
	// =========================================================================
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	kmsCfg := app.Config.ToKMSServiceConfig()

	log.Info("🔍 Sprawdzanie stanu serwisu KMS...")
	if err := kms.HealthCheck(ctx, kmsCfg); err != nil {
		log.Error("❌ KMS Health Check nie powiódł się", "error", err)
		os.Exit(1)
	}

	// 3a. Pobranie klucza HMAC do komunikacji z Gatewayem
	log.Info("🔑 Pobieranie klucza HMAC dla Gateway z KMS...")
	gatewayHmacKey, version, err := kms.FetchSymmetricKeyWithVersion(ctx, kmsCfg, "hmac-gateway-identity", "HmacSha256")
	if err != nil {
		log.Error("❌ Nie udało się pobrać klucza HMAC Gateway z KMS", "error", err)
		os.Exit(1)
	}
	app.Config.Internal.HMACSecret = string(gatewayHmacKey)
	log.Info("✅ Klucz HMAC Gateway pobrany pomyślnie z KMS", "version", version)

	// 3b. Pobranie klucza HMAC do Blind Indexu PESEL!
	log.Info("🔑 Pobieranie klucza HMAC do PESEL Blind Index z KMS...")
	peselHmacKey, peselKeyVersion, err := kms.FetchSymmetricKeyWithVersion(ctx, kmsCfg, "identity-pesel-blind-index", "HmacSha256")
	if err != nil {
		log.Error("❌ Nie udało się pobrać klucza HMAC dla PESEL z KMS", "error", err)
		os.Exit(1)
	}
	log.Info("✅ Klucz HMAC dla PESEL pobrany pomyślnie z KMS", "version", peselKeyVersion)

	// =========================================================================
	// 4. DI CONTAINER & ROUTER
	// =========================================================================
	// Przekazujesz eventPublisher do DI Container (musisz dopisać pole/argument w di.BuildContainer)
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
