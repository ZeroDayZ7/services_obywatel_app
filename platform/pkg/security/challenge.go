package security

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/zerodayz7/platform/pkg/shared"
)

var (
	ErrInvalidSignature = errors.New("security: invalid cryptographic signature")
	ErrInvalidEncoding  = errors.New("security: failed to decode base64 data")
)

func ConstructChallengePayload(challengeBytes []byte, domain string) []byte {
	prefix := []byte(domain + ":")

	payload := make([]byte, len(prefix)+len(challengeBytes))
	copy(payload, prefix)
	copy(payload[len(prefix):], challengeBytes)

	return payload
}

// VerifyEd25519Challenge weryfikuje podpis kluczem Ed25519.
func VerifyEd25519Challenge(pubKeyBytes []byte, challengeB64 string, signatureB64 string, domain string) error {
	log := shared.GetLogger()

	challengeBytes, err := base64.StdEncoding.DecodeString(challengeB64)
	if err != nil {
		return fmt.Errorf("%w: challenge", ErrInvalidEncoding)
	}

	sigBytes, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil {
		return fmt.Errorf("%w: signature", ErrInvalidEncoding)
	}

	if len(pubKeyBytes) != ed25519.PublicKeySize {
		log.WarnMap("[VerifyEd25519Challenge] Błędna długość klucza publicznego", map[string]any{
			"got_len":      len(pubKeyBytes),
			"expected_len": ed25519.PublicKeySize,
		})
		return errors.New("security: invalid public key length for Ed25519")
	}

	pubKey := ed25519.PublicKey(pubKeyBytes)
	payload := ConstructChallengePayload(challengeBytes, domain)

	// LOGI DIAGNOSTYCZNE
	log.DebugMap("🔍 [DEBUG GO - STEP 2]", map[string]any{
		"domain":         domain,
		"challenge_b64":  challengeB64,
		"challenge_hex":  fmt.Sprintf("%x", challengeBytes),
		"payload_hex":    fmt.Sprintf("%x", payload),
		"payload_string": string(payload),
		"pubkey_hex":     fmt.Sprintf("%x", pubKeyBytes),
		"pubkey_len":     len(pubKeyBytes),
		"signature_b64":  signatureB64,
		"signature_hex":  fmt.Sprintf("%x", sigBytes),
		"signature_len":  len(sigBytes),
	})

	if !ed25519.Verify(pubKey, payload, sigBytes) {
		log.Warn("[DEBUG GO] ed25519.Verify zwrócił FALSE!")
		return ErrInvalidSignature
	}

	log.Info("✅ [DEBUG GO] ed25519.Verify powiodło się!")
	return nil
}
