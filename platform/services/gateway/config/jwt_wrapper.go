package config

import (
	"slices"

	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/types"
)

var SkipJWT = false

func JWTMiddlewareWithExclusions() fiber.Handler {
	if SkipJWT {
		return func(c *fiber.Ctx) error {
			return c.Next()
		}
	}

	jwtHandler := jwtware.New(NewJWTConfig())
	return func(c *fiber.Ctx) error {
		if slices.Contains(types.PublicPaths, c.Path()) {
			return c.Next()
		}
		return jwtHandler(c)
	}
}
