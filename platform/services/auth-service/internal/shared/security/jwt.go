package security

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ------------------- ACCESS TOKEN (JWT) -------------------

func GenerateJWT(claims jwt.MapClaims, secret string, ttl time.Duration) (string, error) {
	claims["exp"] = jwt.NewNumericDate(time.Now().Add(ttl))
	claims["iat"] = jwt.NewNumericDate(time.Now())

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ValidateJWT(tokenString, secret string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
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

// ------------------- REFRESH TOKEN (LOSOWY) -------------------

func GenerateRefreshToken() (string, error) {
	return GenerateRandomToken(64)
}
