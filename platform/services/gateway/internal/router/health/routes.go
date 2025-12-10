package health

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/shared"
)

func RegisterRoutes(app *fiber.App, checker *Checker) {
	healthGroup := app.Group("/health")
	healthGroup.Use(shared.NewLimiter("health"))
	healthGroup.Get("/", checker.Handler)
}
