package security

import (
	"context"
	"fmt"

	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/kms"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/websocket-service/internal/config"
)

func LoadSecurityKeys(ctx context.Context, app *config.Config, keyStore *httpserver.KeyStore) error {
	log := shared.GetLogger()
	kmsCfg := app.ToKMSServiceConfig()

	log.Info("🔍 Sprawdzanie stanu serwisu KMS...")
	if err := kms.HealthCheck(ctx, kmsCfg); err != nil {
		return fmt.Errorf("KMS Health Check failed: %w", err)
	}

	for senderID, keyTarget := range app.HMAC.TargetKeys {
		keyBytes, version, err := kms.FetchSymmetricKeyWithVersion(
			ctx,
			kmsCfg,
			keyTarget.TargetKey,
			1,
			keyTarget.Algorithm,
		)
		if err != nil {
			return fmt.Errorf("failed to fetch HMAC key for sender %s (target: %s): %w", senderID, keyTarget.TargetKey, err)
		}

		keyStore.SetKey(senderID, keyBytes, uint32(version))
		log.Info("✅ Klucz HMAC załadowany do KeyStore",
			"alias", senderID,
			"target", keyTarget.TargetKey,
			"algorithm", keyTarget.Algorithm,
			"version", version,
		)
	}

	return nil
}
