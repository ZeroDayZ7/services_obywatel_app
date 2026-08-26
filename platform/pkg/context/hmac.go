package context

import (
	"crypto/hmac"
	"encoding/base64"

	"github.com/zerodayz7/platform/pkg/crypto"
)

// SignHMAC generuje podpis HMAC zarejestrowany w formacie Base64.
// #region SignHMAC
func SignHMAC(payload, key []byte) string {
	return base64.StdEncoding.EncodeToString(crypto.ComputeHMAC256(payload, key))
}

// VerifyHMAC sprawdza podpis w czasie stałym (constant-time), dekodując przychodzący podpis z Base64.
// #region VerifyHMAC
func VerifyHMAC(payload []byte, signature string, key []byte) bool {
	providedMAC, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false
	}

	expectedMAC := crypto.ComputeHMAC256(payload, key)

	return hmac.Equal(expectedMAC, providedMAC)
}
