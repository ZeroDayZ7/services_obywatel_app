package config

import (
	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/shared"
)

// NewJWTConfig — konfiguracja middleware JWT dla Fiber z użyciem Ed25519 (EdDSA)
func NewJWTConfig() jwtware.Config {
	return jwtware.Config{
		SigningKey: jwtware.SigningKey{
			JWTAlg: "EdDSA",
			Key:    AppConfig.JWT.AccessPublicKey,
		},
		ContextKey:   "user",
		TokenLookup:  "header:Authorization,cookie:access_token",
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
