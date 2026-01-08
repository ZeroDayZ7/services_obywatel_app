// pkg/shared/crypto.go
package shared

import (
	"crypto/ed25519"
	"encoding/base64"
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
