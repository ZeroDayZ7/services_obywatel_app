package security

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/zerodayz7/platform/pkg/kms"
)

// ------------------- ACCESS TOKEN (JWT VIA KMS) -------------------

// GenerateJWTViaKMS tworzy i podpisuje token JWT zdalnie w KMS bez pobierania klucza prywatnego.
func GenerateJWTViaKMS(ctx context.Context, kmsCfg kms.Config, targetService string, claims jwt.MapClaims, ttl time.Duration) (string, error) {
	claims["exp"] = jwt.NewNumericDate(time.Now().Add(ttl))
	claims["iat"] = jwt.NewNumericDate(time.Now())

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)

	// Pobieramy niepodpisany ciąg "header.payload"
	signingString, err := token.SigningString()
	if err != nil {
		return "", fmt.Errorf("security: failed to get jwt signing string: %w", err)
	}

	// Wysyłamy niepodpisany ciąg do KMS
	sigBytes, _, err := kms.SignData(ctx, kmsCfg, targetService, kms.DefaultAlgorithm, []byte(signingString))
	if err != nil {
		return "", fmt.Errorf("security: remote jwt signing failed: %w", err)
	}

	// JWT wymaga podpisu w formacie RawURLEncoding (bez paddingu '=' oraz z kodowaniem URL-safe)
	sigB64 := base64.RawURLEncoding.EncodeToString(sigBytes)

	return fmt.Sprintf("%s.%s", signingString, sigB64), nil
}

func ValidateJWT(tokenString string, pubKey ed25519.PublicKey) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodEdDSA.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return pubKey, nil
	})
}
