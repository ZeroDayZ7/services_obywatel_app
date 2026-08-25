package main

import (
	"context"
	"os"

	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/kms"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/services/identity-service/config"
)

// LoadSecurityKeys ładuje wszystkie klucze kryptograficzne z KMS do KeyStore
//#region LoadSecurityKeys
func LoadSecurityKeys(ctx context.Context, app *config.App, keyStore *httpserver.KeyStore) {
	log := shared.GetLogger()
	kmsCfg := app.Config.ToKMSServiceConfig()

	log.Info("🔍 Sprawdzanie stanu serwisu KMS...")
	if err := kms.HealthCheck(ctx, kmsCfg); err != nil {
		log.Error("❌ KMS jest niedostępny podczas inicjalizacji", "error", err)
		os.Exit(1)
	}

	// Wszystkie klucze (TargetKeys, TrustedSenders, InternalKeys) są pobierane z jednej metody konfigu
	for alias, target := range app.Config.GetAllSecurityKeys() {
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

	if _, _, ok := keyStore.GetKey("rabbitmq"); !ok {
		log.Error("❌ Brak klucza RabbitMQ w KeyStore po załadowaniu z KMS")
		os.Exit(1)
	}
}
