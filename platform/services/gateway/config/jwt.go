package config

import (
	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/shared"
)

// NewJWTConfig — konfiguracja middleware JWT dla Fiber
func NewJWTConfig() jwtware.Config {
	var signingKey jwtware.SigningKey

	if len(AppConfig.JWT.AccessPublicKey) > 0 {
		signingKey = jwtware.SigningKey{
			JWTAlg: "EdDSA",
			Key:    AppConfig.JWT.AccessPublicKey,
		}
	} else {
		signingKey = jwtware.SigningKey{
			JWTAlg: jwtware.HS256,
			Key:    []byte(AppConfig.JWT.AccessSecret),
		}
	}

	return jwtware.Config{
		SigningKey:   signingKey,
		ContextKey:   "user",
		TokenLookup:  "header:Authorization",
		AuthScheme:   "Bearer",
		ErrorHandler: jwtErrorHandler,
	}
}

// jwtErrorHandler — standardowa obsługa błędów JWT
func jwtErrorHandler(c *fiber.Ctx, err error) error {
	log := shared.GetLogger()
	log.WarnObj("JWT error", err.Error())
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
		"error": "Unauthorized or invalid token",
	})
}
