package envelope

import (
	"context"
	"fmt"

	"github.com/zerodayz7/platform/pkg/crypto"
	"github.com/zerodayz7/platform/pkg/kms"
)

type EncryptedPayload struct {
	EncryptedData []byte // Zaszyfrowane dane dokumentu
	EncryptedDEK  []byte // Zaszyfrowany klucz DEK
	KeyVersion    int    // Wersja klucza KEK z KMS
}

type EnvelopeCryptor struct {
	kmsCfg kms.Config
}

func NewEnvelopeCryptor(kmsCfg kms.Config) *EnvelopeCryptor {
	return &EnvelopeCryptor{kmsCfg: kmsCfg}
}

// #region Seal
// Seal wykonuje pełny proces szyfrowania kopertowego:
// 1. Generuje losowy DEK (32 bajty)
// 2. Szyfruje nim plaintext (AES-256-GCM)
// 3. Szyfruje DEK w KMS za pomocą KEK o podanym keyAlias
func (e *EnvelopeCryptor) Seal(ctx context.Context, keyAlias string, plaintext []byte) (*EncryptedPayload, error) {
	// 1. Generuj świeży DEK
	dek, err := crypto.GenerateDEK(32)
	if err != nil {
		return nil, err
	}

	// 2. Szyfruj dane w lokalnej pamięci
	encryptedData, err := crypto.EncryptAESGCM(plaintext, dek)
	if err != nil {
		return nil, fmt.Errorf("envelope: failed to encrypt payload: %w", err)
	}

	// 3. Zaszyfruj DEK w KMS i pobierz wersję klucza
	encryptedDEK, keyVersion, err := kms.EncryptDEK(ctx, e.kmsCfg, keyAlias, dek)
	if err != nil {
		return nil, fmt.Errorf("envelope: failed to encrypt DEK via KMS: %w", err)
	}

	return &EncryptedPayload{
		EncryptedData: encryptedData,
		EncryptedDEK:  encryptedDEK,
		KeyVersion:    keyVersion,
	}, nil
}

// #endregion

// #region Unseal
// Unseal wykonuje proces odszyfrowywania kopertowego:
// 1. Wysyła EncryptedDEK do KMS w celu uzyskania surowego DEK
// 2. Odszyfrowuje dane w lokalnej pamięci przy użyciu surowego DEK
func (e *EnvelopeCryptor) Unseal(ctx context.Context, keyAlias string, payload EncryptedPayload) ([]byte, error) {
	// 1. Odszyfruj DEK w KMS
	plaintextDEK, err := kms.DecryptDEK(ctx, e.kmsCfg, keyAlias, payload.EncryptedDEK)
	if err != nil {
		return nil, fmt.Errorf("envelope: failed to decrypt DEK via KMS: %w", err)
	}

	// 2. Odszyfruj dane lokalnie
	plaintext, err := crypto.DecryptAESGCM(payload.EncryptedData, plaintextDEK)
	if err != nil {
		return nil, fmt.Errorf("envelope: failed to decrypt payload: %w", err)
	}

	return plaintext, nil
}

// #endregion
