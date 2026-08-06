package security

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
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

func ValidateJWT(tokenString string, pubKey ed25519.PublicKey) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		// Sprawdzamy, czy metoda to Ed25519 (EdDSA)
		if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, errors.New("unexpected signing method")
		}
		// Zwracamy KLUCZ PUBLICZNY do weryfikacji podpisu
		return pubKey, nil
	})
}

// ------------------- REFRESH TOKEN (LOSOWY) -------------------

func GenerateRandomToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(b), nil
}

func GenerateRefreshToken() (string, error) {
	return GenerateRandomToken(64)
}
