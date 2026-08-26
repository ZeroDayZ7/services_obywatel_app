package crypto // lub package shared

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

// ComputeHMAC256 oblicza surowy podpis HMAC-SHA256 w postaci []byte.
func ComputeHMAC256(payload, key []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(payload)
	return h.Sum(nil)
}

// ComputeHMAC256Hex zwraca podpis HMAC-SHA256 jako ciąg znaków w formacie Hex (np. dla RabbitMQ / HTTP).
func ComputeHMAC256Hex(payload, key []byte) string {
	return hex.EncodeToString(ComputeHMAC256(payload, key))
}

// ComputeHMAC256Base64 zwraca podpis HMAC-SHA256 zakodowany w Base64.
func ComputeHMAC256Base64(payload, key []byte) string {
	return base64.StdEncoding.EncodeToString(ComputeHMAC256(payload, key))
}

// VerifyHMAC256 bezpiecznie porównuje oczekiwany MAC z wyliczonym (odporne na timing attack).
func VerifyHMAC256(payload, expectedMAC, key []byte) bool {
	mac := ComputeHMAC256(payload, key)
	return hmac.Equal(mac, expectedMAC)
}
