package shared

import (
	"crypto/ed25519"
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
