package router

import (
	"github.com/zerodayz7/platform/pkg/errors"

	"github.com/zerodayz7/platform/pkg/shared"

	"github.com/gofiber/fiber/v2"
)

// SetupFallbackHandlers - obsługa 404 i favicon
func SetupFallbackHandlers(app *fiber.App) {
	app.Get("/favicon.ico", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	app.Use(func(c *fiber.Ctx) error {
		log := shared.GetLogger()
		log.WarnMap("404 Not Found", map[string]any{
			"path":      c.Path(),
			"method":    c.Method(),
			"ip":        c.IP(),
			"userAgent": c.Get("User-Agent"),
			"requestID": c.Locals("requestid"),
		})
		return errors.SendAppError(c, errors.ErrNotFound)
	})
}
