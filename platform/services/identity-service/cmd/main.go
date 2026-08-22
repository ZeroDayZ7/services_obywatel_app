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

func LoadSecurityKeys(ctx context.Context, app *config.App, keyStore *httpserver.KeyStore) ([]byte, []byte, error) {
	log := shared.GetLogger()
	kmsCfg := app.Config.ToKMSServiceConfig()

	log.Info("🔍 Sprawdzanie stanu serwisu KMS...")
	if err := kms.HealthCheck(ctx, kmsCfg); err != nil {
		return nil, nil, err
	}

	for senderID, targetKey := range app.Config.HMAC.TargetKeys {
		hmacKey, version, err := kms.FetchSymmetricKeyWithVersion(ctx, kmsCfg, targetKey, "HmacSha256")
		if err != nil {
			return nil, nil, err
		}
		keyStore.SetKey(senderID, hmacKey, version)
		log.Info("✅ Klucz HMAC HTTP załadowany", "service", senderID, "version", version)
	}

	for senderID, targetKey := range app.Config.RabbitConsumers.TrustedSenders {
		hmacKey, version, err := kms.FetchSymmetricKeyWithVersion(ctx, kmsCfg, targetKey, "HmacSha256")
		if err != nil {
			return nil, nil, err
		}
		keyStore.SetKey(senderID, hmacKey, version)
		log.Info("✅ Klucz HMAC Consumer RabbitMQ załadowany", "service", senderID, "version", version)
	}

	peselHmacKey, peselKeyVersion, err := kms.FetchSymmetricKeyWithVersion(ctx, kmsCfg, "identity-pesel-blind-index", "HmacSha256")
	if err != nil {
		return nil, nil, err
	}
	log.Info("✅ Klucz HMAC dla PESEL pobrany pomyślnie z KMS", "version", peselKeyVersion)

	rabbitHMACKey, rabbitKeyVersion, err := kms.FetchSymmetricKeyWithVersion(ctx, kmsCfg, "hmac-identity-rabbitmq", "HmacSha256")
	if err != nil {
		return nil, nil, err
	}
	log.Info("✅ Klucz HMAC dla RabbitMQ pobrany pomyślnie z KMS", "version", rabbitKeyVersion)

	return peselHmacKey, rabbitHMACKey, nil
}

func main() {
	log := shared.GetLogger()

	app, closeDB := config.InitApp()
	defer closeDB()

	keyStore := httpserver.NewKeyStore()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	peselHmacKey, rabbitHMACKey, err := LoadSecurityKeys(ctx, app, keyStore)
	if err != nil {
		log.Error("❌ Nie udało się załadować kluczy bezpieczeństwa z KMS", "error", err)
		os.Exit(1)
	}

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

	container := di.BuildContainer(app, eventPublisher, peselHmacKey, app.Config.ToKMSServiceConfig(), keyStore)

	r := router.NewRouter(container)
	server := &http.Server{
		Addr:         ":" + app.Config.Server.Port,
		Handler:      r,
		ReadTimeout:  app.Config.Server.ReadTimeout,
		WriteTimeout: app.Config.Server.WriteTimeout,
		IdleTimeout:  app.Config.Server.IdleTimeout,
	}

	if err := httpserver.Run(server, app.Config.Shutdown); err != nil {
		log.Error("Server forced shutdown with error", "error", err)
	}
}
