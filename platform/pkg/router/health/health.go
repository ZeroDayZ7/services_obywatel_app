package health

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/shared"
)

func RegisterRoutes(app *fiber.App, checker *Checker) {
	app.Get("/health", shared.NewLimiter("health", nil), checker.Handler)
}
