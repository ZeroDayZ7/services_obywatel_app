package security

import (
	"context"
	"fmt"

	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/kms"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/messaging-service/config"
)

func LoadSecurityKeys(ctx context.Context, cfg *config.Config, keyStore *httpserver.KeyStore) ([]byte, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	if keyStore == nil {
		return nil, fmt.Errorf("keystore is nil")
	}

	log := shared.GetLogger()
	kmsCfg := cfg.ToKMSServiceConfig()

	log.Info("🔍 Sprawdzanie stanu serwisu KMS...")
	if err := kms.HealthCheck(ctx, kmsCfg); err != nil {
		log.Error("❌ KMS jest niedostępny podczas inicjalizacji", "error", err)
		return nil, fmt.Errorf("kms health check: %w", err)
	}

	for alias, target := range cfg.GetAllSecurityKeys() {
		// Jeśli RabbitMQ jest wyłączony, pomijamy pobieranie jego klucza z KMS
		if alias == "rabbitmq" && !cfg.RabbitMQEnabled {
			log.Warn("RabbitMQ jest wyłączony. Pomijam pobieranie klucza RabbitMQ z KMS.")
			continue
		}

		keyBytes, version, err := kms.FetchSymmetricKeyWithVersion(ctx, kmsCfg, target.TargetKey, 1, target.Algorithm)
		if err != nil {
			log.Error("❌ Nie udało się pobrać klucza z KMS",
				"alias", alias,
				"target", target.TargetKey,
				"algorithm", target.Algorithm,
				"error", err,
			)
			return nil, fmt.Errorf("fetch key %s (%s): %w", alias, target.TargetKey, err)
		}

		keyStore.SetKey(alias, keyBytes, uint32(version))
		log.Info("✅ Klucz załadowany do KeyStore",
			"alias", alias,
			"target", target.TargetKey,
			"algorithm", target.Algorithm,
			"version", version,
		)
	}

	// Pobieramy klucz rabbitmq bezpośrednio z keystore (jeśli włączony) na potrzeby main.go
	var rabbitHMACKey []byte
	if cfg.RabbitMQEnabled {
		if key, _, ok := keyStore.GetKey("rabbitmq"); ok {
			rabbitHMACKey = key
		}
	}

	return rabbitHMACKey, nil
}
