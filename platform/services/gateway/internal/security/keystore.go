package security

import (
	"context"
	"fmt"

	"github.com/zerodayz7/platform/pkg/kms"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/gateway/config"
	"github.com/zerodayz7/platform/services/gateway/internal/hmac"
)

// LoadSecurityKeys obsługuje połączenie z KMS, health-check oraz pobieranie kluczy JWT, HMAC serwisów i opcjonalnie RabbitMQ.
//#region LoadSecurityKeys
func LoadSecurityKeys(ctx context.Context, appCfg *config.Config, keyStore *hmac.GatewayKeyStore) ([]byte, error) {
	log := shared.GetLogger()
	kmsCfg := appCfg.ToKMSServiceConfig()

	log.Info("🔍 Sprawdzanie stanu serwisu KMS...")
	if err := kms.HealthCheck(ctx, kmsCfg); err != nil {
		return nil, fmt.Errorf("KMS Health Check failed: %w", err)
	}

	// 1. Pobranie klucza publicznego do weryfikacji tokenów JWT użytkowników
	log.Info("🔑 Pobieranie klucza publicznego JWT z KMS...")
	pubKey, err := kms.FetchPublicKey(ctx, kmsCfg, "shared-jwt")
	if err != nil {
		return nil, fmt.Errorf("KMS Public Key fetch failed: %w", err)
	}
	appCfg.JWT.AccessPublicKey = pubKey
	log.Info("✅ KMS Public Key loaded successfully")

	// 2. Pobranie dedykowanych kluczy HMAC dla poszczególnych mikrousług
	for serviceID, target := range appCfg.HMAC.TargetKeys {
		hmacKey, version, err := kms.FetchSymmetricKeyWithVersion(ctx, kmsCfg, target.TargetKey, 1, target.Algorithm)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch HMAC key for service %s (target: %s): %w", serviceID, target.TargetKey, err)
		}

		keyStore.SetKey(serviceID, hmacKey, uint32(version))
		log.Info("✅ Klucz HMAC załadowany", "service", serviceID, "version", version)
	}

	// 3. Pobranie klucza HMAC dla RabbitMQ Publishera w Gateway (tylko jeśli RabbitMQ jest włączony)
	var publisherHMACKey []byte
	if appCfg.RabbitMQ.Enabled {
		log.Info("RabbitMQ jest ENABLED. Pobieranie klucza HMAC dla Gateway Publisher...")
		rabbitTarget := appCfg.HMAC.RabbitMQKey
		publisherHMACKey, _, err = kms.FetchSymmetricKeyWithVersion(
			ctx,
			kmsCfg,
			rabbitTarget.TargetKey,
			1,
			rabbitTarget.Algorithm,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch RabbitMQ HMAC key for Gateway from KMS: %w", err)
		}
	}

	return publisherHMACKey, nil
}
