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
func LoadSecurityKeys(ctx context.Context, app *config.App, keyStore *httpserver.KeyStore) {
	log := shared.GetLogger()
	kmsCfg := app.Config.ToKMSServiceConfig()

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
	for senderID, keyTarget := range app.Config.HMAC.TargetKeys {
		loadKey(senderID, keyTarget)
	}

	// 2. Zaufani nadawcy RabbitMQ
	for senderID, keyTarget := range app.Config.RabbitConsumers.TrustedSenders {
		loadKey(senderID, keyTarget)
	}

	// 3. Klucze wewnętrzne serwisu
	// Jeśli dodasz nowy klucz w configu, musisz go dopisać również tutaj (chyba że przeniesiesz to do slice w configu)
	internalKeys := map[string]config.KeyTarget{
		"pesel":      app.Config.HMAC.PeselKey,
		"phone":      app.Config.HMAC.PhoneKey,
		"email":      app.Config.HMAC.EmailKey,
		"puk":        app.Config.HMAC.PukKey,
		"rabbitmq":   app.Config.HMAC.RabbitMQKey,
		"audit":      app.Config.HMAC.AuditKey,
		"agreements": app.Config.HMAC.AgreementsKey,
	}

	for alias, keyTarget := range internalKeys {
		loadKey(alias, keyTarget)
	}

	if _, _, ok := keyStore.GetKey("rabbitmq"); !ok {
		log.Error("❌ Brak klucza RabbitMQ w KeyStore po załadowaniu z KMS")
		os.Exit(1)
	}
}
