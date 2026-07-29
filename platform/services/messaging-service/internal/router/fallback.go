package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/shared"
)

func SetupFallbackHandlers(app *fiber.App) {
	app.Get("/favicon.ico", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	app.Use(func(c *fiber.Ctx) error {
		shared.GetLogger().WarnMap("404 - not found", map[string]any{
			"path":   c.Path(),
			"method": c.Method(),
		})
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Not found",
		})
	})
}
