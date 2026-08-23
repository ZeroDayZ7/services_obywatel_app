package main

import (
	"context"

	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/kms"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/auth-service/config"
)

func LoadSecurityKeys(ctx context.Context, app *config.Config, keyStore *httpserver.KeyStore) ([]byte, error) {
	log := shared.GetLogger()
	kmsCfg := app.ToKMSServiceConfig()

	log.Info("🔍 Sprawdzanie stanu serwisu KMS...")
	if err := kms.HealthCheck(ctx, kmsCfg); err != nil {
		return nil, err
	}

	if app.JWT.SigningMode == "local" {
		log.Info("🔑 [MODE: LOCAL] Pobieranie klucza prywatnego JWT z KMS do pamięci serwisu...")
		// Usunięto zbędny argument "shared-jwt" (funkcja przyjmuje tylko ctx i cfg)
		privKey, err := kms.FetchAuthPrivateKey(ctx, kmsCfg)
		if err != nil {
			return nil, err
		}
		app.JWT.AccessPrivateKey = privKey
		log.Info("✅ Pomyślnie pobrano i załadowano klucz prywatny JWT do pamięci")
	} else {
		log.Info("🛡️ [MODE: KMS] Tokeny będą podpisywane zdalnie przez API KMS")
	}

	for senderID, targetKey := range app.HMAC.TargetKeys {
		// Poprawiono argumenty: FetchSymmetricKeyWithVersion oczekuje (ctx, cfg, purpose, version int)
		// Załóżmy, że pobieramy domyślnie wersję 1 lub przekazujesz numer wersji z configu. Tutaj użyto wersji 1 jako przykładu.
		hmacKey, version, err := kms.FetchSymmetricKeyWithVersion(ctx, kmsCfg, targetKey, 1)
		if err != nil {
			return nil, err
		}
		// Rzutowanie version (int) na uint32 wymagane przez SetKey
		keyStore.SetKey(senderID, hmacKey, uint32(version))
		log.Info("✅ Klucz HMAC załadowany", "service", senderID, "version", version)
	}

	for senderID, targetKey := range app.RabbitConsumers.TrustedSenders {
		hmacKey, version, err := kms.FetchSymmetricKeyWithVersion(ctx, kmsCfg, targetKey, 1)
		if err != nil {
			return nil, err
		}
		keyStore.SetKey(senderID, hmacKey, uint32(version))
		log.Info("✅ Klucz HMAC Consumer RabbitMQ załadowany", "service", senderID, "version", version)
	}

	rabbitHMACKey, rabbitKeyVersion, err := kms.FetchSymmetricKeyWithVersion(ctx, kmsCfg, "hmac-auth-rabbitmq", 1)
	if err != nil {
		return nil, err
	}
	log.Info("✅ Klucz HMAC dla RabbitMQ pobrany pomyślnie z KMS", "version", rabbitKeyVersion)

	return rabbitHMACKey, nil
}
