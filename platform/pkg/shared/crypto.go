// pkg/shared/crypto.go
package shared

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
)

var (
	ErrInvalidCiphertext = errors.New("ciphertext is too short or corrupted")
	ErrInvalidKeySize    = errors.New("invalid key size for AES encryption")
)

//#region VerifyEd25519Signature
// VerifyEd25519Signature sprawdza, czy podpis Base64 jest poprawny dla podanych bajtów wiadomości i klucza publicznego Base64.
func VerifyEd25519Signature(publicKeyBase64 string, message []byte, signatureBase64 string) bool {
	pubKey, err := base64.StdEncoding.DecodeString(publicKeyBase64)
	if err != nil || len(pubKey) != ed25519.PublicKeySize {
		return false
	}

	sig, err := base64.StdEncoding.DecodeString(signatureBase64)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return false
	}

	return ed25519.Verify(pubKey, message, sig)
}

//#region Encrypt
// Encrypt szyfruje tekst jawny przy użyciu AES-GCM z losowym wektorem inicjalizacyjnym (nonce).
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

//#region Decrypt
// Decrypt odszyfrowuje dane AES-GCM, wyciągając nonce z pierwszych bajtów szyfrogramu.
func Decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrInvalidCiphertext
	}

	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, actualCiphertext, nil)
}

//#region ComputeHMACSHA256
// ComputeHMACSHA256 oblicza keyed-hash (HMAC-SHA256) dla podanych danych i klucza,
// zwracając wynik w postaci ciągu szesnastkowego (hex).
func ComputeHMACSHA256(data []byte, key []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

//#region ComputeHMACSHA256Base64
// ComputeHMACSHA256Base64 oblicza HMAC-SHA256 i zwraca wynik zakodowany w Base64.
func ComputeHMACSHA256Base64(data []byte, key []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

//#region VerifyHMACSHA256
// VerifyHMACSHA256 bezpiecznie (w czasie stałym) weryfikuje podpis HMAC-SHA256 zapobiegając atakom typu Timing Attack.
func VerifyHMACSHA256(data []byte, expectedMAC []byte, key []byte) bool {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	actualMAC := h.Sum(nil)
	return subtle.ConstantTimeCompare(actualMAC, expectedMAC) == 1
}
