package config

import (
	"slices"

	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
)

// PublicPaths - ścieżki, które NIE wymagają autoryzacji JWT
var PublicPaths = []string{
	"/auth/login",
	"/auth/register",
	"/auth/refresh",
	"/auth/2fa-verify",
	"/auth/reset/verify",
	"/auth/reset/send",
	"/auth/reset/final",
	"/auth/2fa-resend",
	"/health",
}

var SkipJWT = false

// JWTMiddlewareWithExclusions — middleware JWT z obsługą wyjątków (publicznych tras)
func JWTMiddlewareWithExclusions() fiber.Handler {
	if SkipJWT {
		// w testach omijamy JWT
		return func(c *fiber.Ctx) error {
			return c.Next()
		}
	}

	jwtHandler := jwtware.New(NewJWTConfig())
	return func(c *fiber.Ctx) error {
		if slices.Contains(PublicPaths, c.Path()) {
			return c.Next()
		}
		return jwtHandler(c)
	}
}
