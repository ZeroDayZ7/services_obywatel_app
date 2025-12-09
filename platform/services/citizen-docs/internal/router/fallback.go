package router

import (
	"github.com/zerodayz7/platform/services/citizen-docs/internal/shared/logger"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func SetupFallbackHandlers(app *fiber.App) {
	app.Get("/favicon.ico", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	app.Use(func(c *fiber.Ctx) error {
		logger.GetLogger().Warn("404 - not found", zap.String("path", c.Path()), zap.String("method", c.Method()))
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Not found",
		})
	})
}
