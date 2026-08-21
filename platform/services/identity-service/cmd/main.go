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

	// Instancja KeyStore do przechowywania kluczy dla akceptowanych nadawców
	keyStore := httpserver.NewKeyStore()

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

	// 2a. Pobieranie kluczy HMAC dla dopuszczonych nadawców HTTP (Gateway, BFF)
	for senderID, targetKey := range app.Config.HMAC.TargetKeys {
		hmacKey, version, err := kms.FetchSymmetricKeyWithVersion(ctx, kmsCfg, targetKey, "HmacSha256")
		if err != nil {
			log.Error("❌ Nie udało się pobrać klucza HMAC z KMS", "sender", senderID, "target_key", targetKey, "error", err)
			os.Exit(1)
		}

		keyStore.SetKey(senderID, hmacKey, version)
		log.Info("✅ Klucz HMAC HTTP załadowany", "service", senderID, "version", version)
	}

	// 2b. Pobieranie kluczy HMAC dla zaufanych nadawców zdarzeń RabbitMQ (np. auth-service)
	for senderID, targetKey := range app.Config.RabbitConsumers.TrustedSenders {
		hmacKey, version, err := kms.FetchSymmetricKeyWithVersion(ctx, kmsCfg, targetKey, "HmacSha256")
		if err != nil {
			log.Error("❌ Nie udało się pobrać klucza HMAC dla Consumer RabbitMQ z KMS", "sender", senderID, "target_key", targetKey, "error", err)
			os.Exit(1)
		}

		keyStore.SetKey(senderID, hmacKey, version)
		log.Info("✅ Klucz HMAC Consumer RabbitMQ załadowany", "service", senderID, "version", version)
	}

	// 2c. Pobranie klucza HMAC do Blind Indexu PESEL
	peselHmacKey, peselKeyVersion, err := kms.FetchSymmetricKeyWithVersion(ctx, kmsCfg, "identity-pesel-blind-index", "HmacSha256")
	if err != nil {
		log.Error("❌ Nie udało się pobrać klucza HMAC dla PESEL z KMS", "error", err)
		os.Exit(1)
	}
	log.Info("✅ Klucz HMAC dla PESEL pobrany pomyślnie z KMS", "version", peselKeyVersion)

	// 2d. Pobranie klucza HMAC dla własnego RabbitMQ Publishera
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
			app.Config.Server.AppName,
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
	// Przekazujemy keyStore do DI Containera (pamiętaj zaktualizować konstruktor BuildContainer!)
	container := di.BuildContainer(app, eventPublisher, peselHmacKey, kmsCfg, keyStore)

	// 5. Budowanie routera na natywnym http.ServeMux
	r := router.NewRouter(container)

	// 6. Konfiguracja instancji serwera HTTP
	srv := &http.Server{
		Addr:         ":" + app.Config.Server.Port,
		Handler:      r,
		ReadTimeout:  app.Config.Server.ReadTimeout,
		WriteTimeout: app.Config.Server.WriteTimeout,
		IdleTimeout:  app.Config.Server.IdleTimeout,
	}

	// 7. Uruchomienie z Graceful Shutdown
	if err := httpserver.Run(srv, app.Config.Shutdown); err != nil {
		log.Error("Server forced shutdown with error", "error", err)
	}
}
