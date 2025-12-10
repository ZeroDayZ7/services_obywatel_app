package security

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/shared"
)

// ------------------- ACCESS TOKEN (JWT) -------------------

func GenerateAccessToken(userID string, cache *redis.Cache, secret string) (string, error) {
	ctx := context.Background()
	var sessionID string

	// Unikalne sessionID
	for {
		sessionID = shared.GenerateUuid()
		exists, err := cache.Exists(ctx, "session:"+sessionID)
		if err != nil {
			return "", err
		}
		if !exists {
			break
		}
	}

	// Zapis do Redis
	err := cache.Set(ctx, "session:"+sessionID, userID)
	if err != nil {
		return "", err
	}

	claims := jwt.MapClaims{
		"sid": sessionID,
		"exp": jwt.NewNumericDate(time.Now().Add(cache.TTL())),
		"iat": jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func ValidateAccessToken(tokenString string, secret string) (*jwt.Token, error) {
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

func GenerateRefreshToken() (string, error) {
	return GenerateRandomToken(64)
}
