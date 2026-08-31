package config

import (
	"slices"
	"strings"

	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/constants"
	"github.com/zerodayz7/platform/pkg/shared"
)

// JWTMiddleware tworzy i konfiguruje middleware JWT.
// Automatycznie ignoruje ścieżki zdefiniowane w constants.PublicPaths.
func JWTMiddleware() fiber.Handler {
	return jwtware.New(jwtware.Config{
		SigningKey: jwtware.SigningKey{
			JWTAlg: "EdDSA",
			Key:    AppConfig.JWT.AccessPublicKey,
		},
		ContextKey:  "user",
		TokenLookup: "header:Authorization,cookie:access_token",
		AuthScheme:  "Bearer",

		Filter: func(c *fiber.Ctx) bool {
			path := strings.TrimRight(c.Path(), "/")

			// Pomijamy ikony przeglądarki i preflight CORS
			if path == "/favicon.ico" || c.Method() == fiber.MethodOptions {
				return true
			}

			// fmt.Printf("[DEBUG JWT Filter] Path: %s | IsPublic: %t\n", path, isPublic)

			if path == "" {
				path = "/"
			}

			return slices.Contains(constants.PublicPaths, path)
		},

		ErrorHandler: jwtErrorHandler,
	})
}

// jwtErrorHandler — standardowa obsługa błędów JWT
func jwtErrorHandler(c *fiber.Ctx, err error) error {
	log := shared.GetLogger()
	log.WarnObj("JWT error", map[string]any{"error": err.Error()}) // Dopasowano do WarnObj przyjmującego mapę/strukturę

	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
		"error": "Unauthorized or invalid token",
	})
}
