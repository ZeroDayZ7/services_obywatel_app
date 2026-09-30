package security

import (
	"context"

	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/kms"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/messaging-service/config"
)

func LoadSecurityKeys(ctx context.Context, app *config.Config, keyStore *httpserver.KeyStore) ([]byte, error) {
	log := shared.GetLogger()
	kmsCfg := app.ToKMSServiceConfig()

	log.Info("🔍 Sprawdzanie stanu serwisu KMS...")
	if err := kms.HealthCheck(ctx, kmsCfg); err != nil {
		return nil, err
	}

	loadKey := func(alias string, target config.KeyTarget) {
		keyBytes, version, err := kms.FetchSymmetricKeyWithVersion(ctx, kmsCfg, target.TargetKey, 1, target.Algorithm)
		if err != nil {
			log.Error("❌ Nie udało się pobrać klucza z KMS",
				"alias", alias,
				"target", target.TargetKey,
				"algorithm", target.Algorithm,
				"error", err,
			)
			return
		}

		keyStore.SetKey(alias, keyBytes, uint32(version))
		log.Info("✅ Klucz załadowany do KeyStore",
			"alias", alias,
			"target", target.TargetKey,
			"algorithm", target.Algorithm,
			"version", version,
		)
	}

	// Klucze dla nadawców zewnętrznych/wewnętrznych HTTP (np. Gateway)
	for senderID, keyTarget := range app.HMAC.TargetKeys {
		loadKey(senderID, keyTarget)
	}

	// 3. Wewnętrzny klucz RabbitMQ dla tego serwisu
	rabbitTarget := app.HMAC.RabbitMQKey
	rabbitHMACKey, version, err := kms.FetchSymmetricKeyWithVersion(ctx, kmsCfg, rabbitTarget.TargetKey, 1, rabbitTarget.Algorithm)
	if err != nil {
		return nil, err
	}
	log.Info("✅ Klucz HMAC dla RabbitMQ pobrany pomyślnie z KMS", "target", rabbitTarget.TargetKey, "version", version)

	return rabbitHMACKey, nil
}
