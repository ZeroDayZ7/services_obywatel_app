package security

import (
	"context"
	"fmt"

	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/kms"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/citizen-docs/config"
)

// LoadSecurityKeys loads all symmetric keys required by the service from KMS into the provided KeyStore.
func LoadSecurityKeys(ctx context.Context, cfg *config.Config, keyStore *httpserver.KeyStore) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if keyStore == nil {
		return fmt.Errorf("keystore is nil")
	}

	log := shared.GetLogger()
	kmsCfg := cfg.ToKMSServiceConfig()

	log.Info("🔍 Sprawdzanie stanu serwisu KMS...")
	if err := kms.HealthCheck(ctx, kmsCfg); err != nil {
		log.Error("❌ KMS jest niedostępny podczas inicjalizacji", "error", err)
		return fmt.Errorf("kms health check: %w", err)
	}

	for alias, target := range cfg.GetAllSecurityKeys() {
		keyBytes, version, err := kms.FetchSymmetricKeyWithVersion(ctx, kmsCfg, target.TargetKey, 1, target.Algorithm)
		if err != nil {
			log.Error("❌ Nie udało się pobrać klucza z KMS",
				"alias", alias,
				"target", target.TargetKey,
				"algorithm", target.Algorithm,
				"error", err,
			)
			return fmt.Errorf("fetch key %s (%s): %w", alias, target.TargetKey, err)
		}

		keyStore.SetKey(alias, keyBytes, uint32(version))
		log.Info("✅ Klucz załadowany do KeyStore",
			"alias", alias,
			"target", target.TargetKey,
			"algorithm", target.Algorithm,
			"version", version,
		)
	}

	return nil
}
