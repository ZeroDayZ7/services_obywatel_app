package security

import (
	"crypto/ed25519"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ------------------- ACCESS TOKEN (JWT) -------------------

func GenerateJWT(claims jwt.MapClaims, privKey ed25519.PrivateKey, ttl time.Duration) (string, error) {
	claims["exp"] = jwt.NewNumericDate(time.Now().Add(ttl))
	claims["iat"] = jwt.NewNumericDate(time.Now())

	// W golang-jwt/v5 właściwa instancja to jwt.SigningMethodEdDSA
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)

	// Podpisujemy KLUCZEM PRYWATNYM
	return token.SignedString(privKey)
}
