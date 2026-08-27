package security

import (
	"context"
	"fmt"
	"time"

	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/kms"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/services/officer-bff/config"
)

// LoadSecurityKeys wykonuje HealthCheck serwisu KMS oraz pobiera i rejestruje
// w KeyStore wszystkie wymagane klucze HMAC zdefiniowane w konfiguracji aplikacji.
func LoadSecurityKeys(ctx context.Context, cfg *config.Config, keyStore *httpserver.KeyStore) error {
	log := shared.GetLogger()

	kmsCfg := cfg.ToKMSServiceConfig()

	log.Info("🔍 Sprawdzanie stanu serwisu KMS...")
	if err := kms.HealthCheck(ctx, kmsCfg); err != nil {
		return fmt.Errorf("KMS Health Check nie powiódł się: %w", err)
	}

	for serviceID, target := range cfg.HMAC.TargetKeys {
		fetchCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		hmacKey, version, err := kms.FetchSymmetricKeyWithVersion(fetchCtx, kmsCfg, target.TargetKey, 1, target.Algorithm)
		cancel()

		if err != nil {
			return fmt.Errorf("nie udało się pobrać klucza HMAC dla serwisu %s (key: %s): %w", serviceID, target.TargetKey, err)
		}

		keyStore.SetKey(serviceID, hmacKey, uint32(version))
		log.Info("✅ Klucz HMAC załadowany", "service", serviceID, "version", version)
	}

	return nil
}