package security

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
)

var (
	ErrInvalidSignature = errors.New("security: invalid cryptographic signature")
	ErrInvalidEncoding  = errors.New("security: failed to decode base64 data")
)

// ConstructChallengePayload buduje sztywno zdefiniowany ciąg bajtów do weryfikacji (Domain Binding).
func ConstructChallengePayload(challengeBytes []byte, domain string) []byte {
	return fmt.Appendf(nil, "%s:%s", domain, string(challengeBytes))
}

// VerifyEd25519Challenge weryfikuje podpis kluczem Ed25519.
func VerifyEd25519Challenge(pubKeyBytes []byte, challengeB64 string, signatureB64 string, domain string) error {
	challengeBytes, err := base64.RawURLEncoding.DecodeString(challengeB64)
	if err != nil {
		return fmt.Errorf("%w: challenge", ErrInvalidEncoding)
	}

	sigBytes, err := base64.RawURLEncoding.DecodeString(signatureB64)
	if err != nil {
		return fmt.Errorf("%w: signature", ErrInvalidEncoding)
	}

	if len(pubKeyBytes) != ed25519.PublicKeySize {
		return errors.New("security: invalid public key length for Ed25519")
	}

	pubKey := ed25519.PublicKey(pubKeyBytes)
	payload := ConstructChallengePayload(challengeBytes, domain)

	if !ed25519.Verify(pubKey, payload, sigBytes) {
		return ErrInvalidSignature
	}

	return nil
}
