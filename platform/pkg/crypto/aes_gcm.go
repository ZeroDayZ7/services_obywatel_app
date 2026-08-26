package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

var (
	ErrInvalidCiphertext = errors.New("crypto: ciphertext is too short or corrupted")
	ErrInvalidKeySize    = errors.New("crypto: invalid key size for AES encryption")
)

// #region GenerateDEK
// GenerateDEK generuje losowy klucz symetryczny o zadanej długości w bajtach (standardowo 32 bajty dla AES-256).
//#region GenerateDEK
func GenerateDEK(size int) ([]byte, error) {
	if size != 16 && size != 24 && size != 32 {
		return nil, ErrInvalidKeySize
	}
	key := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("crypto: failed to generate random DEK: %w", err)
	}
	return key, nil
}
// #endregion

// #region EncryptAESGCM
// EncryptAESGCM szyfruje tekst jawny przy użyciu AES-GCM i pakuje nonce na początek szyfrogramu.
//#region EncryptAESGCM
func EncryptAESGCM(plaintext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("crypto: failed to generate nonce: %w", err)
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}
// #endregion

// #region DecryptAESGCM
// DecryptAESGCM odszyfrowuje dane AES-GCM, wyciągając nonce z pierwszych bajtów szyfrogramu.
//#region DecryptAESGCM
func DecryptAESGCM(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrInvalidCiphertext
	}

	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, actualCiphertext, nil)
}
// #endregion