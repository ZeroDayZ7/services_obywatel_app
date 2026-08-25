// platform/pkg/security/jwt.go
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

// GenerateJWTLocal podpisuje token lokalnie w pamięci mikroserwisu przy użyciu klucza prywatnego Ed25519.
//#region GenerateJWTLocal
func GenerateJWTLocal(claims jwt.MapClaims, ttl time.Duration, privKey ed25519.PrivateKey) (string, error) {
	if len(privKey) == 0 {
		return "", fmt.Errorf("security: private key is empty")
	}

	claims["exp"] = jwt.NewNumericDate(time.Now().Add(ttl))
	claims["iat"] = jwt.NewNumericDate(time.Now())

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	tokenString, err := token.SignedString(privKey)
	if err != nil {
		return "", fmt.Errorf("security: failed to sign jwt locally: %w", err)
	}

	return tokenString, nil
}

// GenerateJWTViaKMS tworzy i podpisuje token JWT zdalnie w KMS bez posiadania klucza prywatnego w pamięci.
//#region GenerateJWTViaKMS
func GenerateJWTViaKMS(ctx context.Context, kmsCfg kms.Config, targetService string, claims jwt.MapClaims, ttl time.Duration) (string, error) {
	claims["exp"] = jwt.NewNumericDate(time.Now().Add(ttl))
	claims["iat"] = jwt.NewNumericDate(time.Now())

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)

	signingString, err := token.SigningString()
	if err != nil {
		return "", fmt.Errorf("security: failed to get jwt signing string: %w", err)
	}

	sigBytes, _, err := kms.SignData(ctx, kmsCfg, targetService, kms.DefaultAlgorithm, []byte(signingString))
	if err != nil {
		return "", fmt.Errorf("security: remote jwt signing failed: %w", err)
	}

	sigB64 := base64.RawURLEncoding.EncodeToString(sigBytes)
	return fmt.Sprintf("%s.%s", signingString, sigB64), nil
}

//#region ValidateJWT
func ValidateJWT(tokenString string, pubKey ed25519.PublicKey) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodEdDSA.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return pubKey, nil
	})
}
