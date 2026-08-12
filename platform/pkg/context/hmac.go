package context

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
)

// ComputeMAC generuje surowy hash HMAC-SHA256 bez kodowania.
func ComputeMAC(payload, secret []byte) []byte {
	h := hmac.New(sha256.New, secret)
	h.Write(payload)
	return h.Sum(nil)
}

// Sign generuje podpis HMAC zarejestrowany w formacie Base64.
func Sign(payload, secret []byte) string {
	return base64.StdEncoding.EncodeToString(ComputeMAC(payload, secret))
}

// Verify sprawdza podpis w czasie stałym (constant-time), dekodując tylko podpis przychodzący.
func Verify(payload []byte, signature string, secret []byte) bool {
	providedMAC, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false
	}

	expectedMAC := ComputeMAC(payload, secret)

	return hmac.Equal(expectedMAC, providedMAC)
}
