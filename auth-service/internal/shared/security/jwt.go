package security

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/zerodayz7/http-server/config"
	"github.com/zerodayz7/http-server/internal/shared"
)

// ------------------- ACCESS TOKEN (JWT) -------------------

func GenerateAccessToken(userID string) (string, error) {
	rdb := config.NewRedisClient()
	ctx := context.Background()

	var sessionID string

	for {
		sessionID = shared.GenerateUuid()

		exists, err := rdb.Exists(ctx, sessionID).Result()
		if err != nil {
			return "", err
		}
		if exists == 0 {
			break
		}
	}

	err := rdb.Set(ctx, "session:"+sessionID, userID, config.AppConfig.SessionTTL).Err()

	if err != nil {
		return "", err
	}

	claims := jwt.MapClaims{
		"sid": sessionID,
		"exp": jwt.NewNumericDate(time.Now().Add(config.AppConfig.JWT.AccessTTL)),
		"iat": jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(config.AppConfig.JWT.AccessSecret))
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func ValidateAccessToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(config.AppConfig.JWT.AccessSecret), nil
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
