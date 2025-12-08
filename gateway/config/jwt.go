package config

import (
	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/http-server/internal/shared/logger"
)

// NewJWTConfig — konfiguracja middleware JWT dla Fiber
func NewJWTConfig() jwtware.Config {
	return jwtware.Config{
		SigningKey:   jwtware.SigningKey{Key: []byte(AppConfig.JWT.AccessSecret)}, // wymagane []byte
		ContextKey:   "user",                                                      // klucz w ctx.Locals()
		TokenLookup:  "header:Authorization",
		AuthScheme:   "Bearer",
		ErrorHandler: jwtErrorHandler,
	}
}

// jwtErrorHandler — standardowa obsługa błędów JWT
func jwtErrorHandler(c *fiber.Ctx, err error) error {
	log := logger.GetLogger()
	log.WarnObj("JWT error", err.Error())
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
		"error": "Unauthorized or invalid token",
	})
}
