// pkg/shared/crypto.go
package shared

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
)

// VerifyEd25519Signature sprawdza czy podpis jest poprawny dla danej wiadomości i klucza publicznego
func VerifyEd25519Signature(publicKeyBase64, message, signatureBase64 string) bool {
	// 1. Dekodujemy klucz publiczny z Base64
	pubKey, err := base64.StdEncoding.DecodeString(publicKeyBase64)
	if err != nil || len(pubKey) != ed25519.PublicKeySize {
		return false
	}

	// 2. Dekodujemy podpis z Base64
	sig, err := base64.StdEncoding.DecodeString(signatureBase64)
	if err != nil {
		return false
	}

	// 3. Weryfikujemy podpis (biblioteka standardowa Go)
	return ed25519.Verify(pubKey, []byte(message), sig)
}

func Encrypt(plaintext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// ComputeHMACSHA256 oblicza keyed-hash (HMAC-SHA256) dla podanych danych i klucza,
// zwracając wynik w postaci ciągu szesnastkowego (hex).
func ComputeHMACSHA256(data []byte, key []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// ComputeHMACSHA256Base64 oblicza HMAC-SHA256 i zwraca wynik zakodowany w Base64.
func ComputeHMACSHA256Base64(data []byte, key []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
