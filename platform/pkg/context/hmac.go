package context

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
)

// ComputeMAC generuje surowy hash HMAC-SHA256 bez kodowania.
func ComputeMAC(payload, key []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(payload)
	return h.Sum(nil)
}

// SignHMAC generuje podpis HMAC zarejestrowany w formacie Base64.
func SignHMAC(payload, key []byte) string {
	return base64.StdEncoding.EncodeToString(ComputeMAC(payload, key))
}

// VerifyHMAC sprawdza podpis w czasie stałym (constant-time), dekodując przychodzący podpis z Base64.
func VerifyHMAC(payload []byte, signature string, key []byte) bool {
	providedMAC, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false
	}

	expectedMAC := ComputeMAC(payload, key)

	return hmac.Equal(expectedMAC, providedMAC)
}
