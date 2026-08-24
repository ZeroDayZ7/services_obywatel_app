package envelope

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/zerodayz7/platform/pkg/crypto"
	"github.com/zerodayz7/platform/pkg/kms"
)

// EncryptedPayload przechowuje zaszyfrowane dane oraz gotowy do zapisu w bazie DEK (z wbudowaną wersją)
type EncryptedPayload struct {
	EncryptedData []byte // Zaszyfrowane dane właściwe (np. dokument)
	EncryptedDEK  []byte // Zaszyfrowany klucz DEK z doklejoną wersją (gotowy do DB)
	KeyVersion    int    // Czysta wersja (gdyby była potrzebna osobno)
}

type EnvelopeCryptor struct {
	kmsCfg kms.Config
}

// #region NewEnvelopeCryptor
func NewEnvelopeCryptor(kmsCfg kms.Config) *EnvelopeCryptor {
	return &EnvelopeCryptor{kmsCfg: kmsCfg}
}

// #region Seal
// Seal wykonuje pełny proces szyfrowania kopertowego i automatycznie pakuje wersję klucza do DEK
func (e *EnvelopeCryptor) Seal(ctx context.Context, keyAlias string, plaintext []byte) (*EncryptedPayload, error) {
	dek, err := crypto.GenerateDEK(32)
	if err != nil {
		return nil, err
	}
	defer kms.ZeroBytes(dek)

	encryptedData, err := crypto.EncryptAESGCM(plaintext, dek)
	if err != nil {
		return nil, fmt.Errorf("envelope: failed to encrypt payload: %w", err)
	}

	rawEncryptedDEK, keyVersion, err := kms.EncryptDEK(ctx, e.kmsCfg, keyAlias, dek)
	if err != nil {
		return nil, fmt.Errorf("envelope: failed to encrypt DEK via KMS: %w", err)
	}

	// Automatyczne pakowanie wersji klucza (4 bajty) + rawEncryptedDEK
	packedDEK := make([]byte, 4+len(rawEncryptedDEK))
	binary.BigEndian.PutUint32(packedDEK[0:4], uint32(keyVersion))
	copy(packedDEK[4:], rawEncryptedDEK)

	return &EncryptedPayload{
		EncryptedData: encryptedData,
		EncryptedDEK:  packedDEK, // Tutaj jest już spakowane!
		KeyVersion:    keyVersion,
	}, nil
}

// #region Unseal
// Unseal automatycznie rozpakowuje wersję klucza z EncryptedDEK i odszyfrowuje dane
func (e *EnvelopeCryptor) Unseal(ctx context.Context, keyAlias string, encryptedData []byte, packedDEK []byte) ([]byte, error) {
	if len(packedDEK) < 4 {
		return nil, fmt.Errorf("envelope: packed DEK too short")
	}

	// Automatyczne wyciągnięcie wersji z pierwszych 4 bajtów
	keyVersion := int(binary.BigEndian.Uint32(packedDEK[0:4]))
	rawEncryptedDEK := packedDEK[4:]

	plaintextDEK, err := kms.DecryptDEK(ctx, e.kmsCfg, keyAlias, rawEncryptedDEK, keyVersion)
	if err != nil {
		return nil, fmt.Errorf("envelope: failed to decrypt DEK via KMS: %w", err)
	}
	defer kms.ZeroBytes(plaintextDEK)

	plaintext, err := crypto.DecryptAESGCM(encryptedData, plaintextDEK)
	if err != nil {
		return nil, fmt.Errorf("envelope: failed to decrypt payload: %w", err)
	}

	return plaintext, nil
}
