package shared

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
)

// #region VerifyEd25519Signature
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

// #region VerifyEd25519SignatureHex
func VerifyEd25519SignatureHex(publicKeyHex string, message []byte, signatureBase64 string) bool {
	pubKey, err := hex.DecodeString(publicKeyHex)
	if err != nil || len(pubKey) != ed25519.PublicKeySize {
		return false
	}

	sig, err := base64.StdEncoding.DecodeString(signatureBase64)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return false
	}

	return ed25519.Verify(pubKey, message, sig)
}

// #region ComputeHMACSHA256
func ComputeHMACSHA256(data []byte, key []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// #region ComputeHMACSHA256Base64
func ComputeHMACSHA256Base64(data []byte, key []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// #region VerifyHMACSHA256
func VerifyHMACSHA256(data []byte, expectedMAC []byte, key []byte) bool {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	actualMAC := h.Sum(nil)
	return subtle.ConstantTimeCompare(actualMAC, expectedMAC) == 1
}

// #region HashSHA256
func HashSHA256(data string) string {
	if data == "" {
		return ""
	}
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}
