package security

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
	"github.com/zerodayz7/http-server/config"
)

// ------------------- ACCESS TOKEN (JWT) -------------------

func ValidateAccessToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(config.AppConfig.JWT.AccessSecret), nil
	})
}
