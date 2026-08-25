package main

import (
	"context"
	"os"

	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/kms"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/citizen-docs/config"
)

// LoadSecurityKeys ładuje wszystkie klucze kryptograficzne z KMS do KeyStore
func LoadSecurityKeys(ctx context.Context, app *config.Config, keyStore *httpserver.KeyStore) {
	log := shared.GetLogger()
	kmsCfg := app.ToKMSServiceConfig()

	log.Info("🔍 Sprawdzanie stanu serwisu KMS...")
	if err := kms.HealthCheck(ctx, kmsCfg); err != nil {
		log.Error("❌ KMS jest niedostępny podczas inicjalizacji", "error", err)
		os.Exit(1)
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
			os.Exit(1)
		}

		keyStore.SetKey(alias, keyBytes, uint32(version))
		log.Info("✅ Klucz załadowany do KeyStore",
			"alias", alias,
			"target", target.TargetKey,
			"algorithm", target.Algorithm,
			"version", version,
		)
	}

	// 1. Zewnętrzni nadawcy (API Gateway, BFF)
	for senderID, keyTarget := range app.HMAC.TargetKeys {
		loadKey(senderID, keyTarget)
	}

	// 2. Wewnętrzne klucze domenowe serwisu
	internalKeys := map[string]config.KeyTarget{
		"pesel": app.HMAC.PeselKey,
	}

	for alias, keyTarget := range internalKeys {
		loadKey(alias, keyTarget)
	}
}
