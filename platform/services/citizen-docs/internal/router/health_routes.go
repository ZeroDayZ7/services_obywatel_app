package router

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/services/citizen-docs/config"
)

func SetupHealthRoutes(app *fiber.App) {
	health := app.Group("/health")

	health.Use(config.NewLimiter("health"))

	// GET /health
	health.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"time":   time.Now().Format("2006-01-02 15:04:05"),
		})
	})
}
