package health

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/shared"
)

//#region RegisterRoutes
func RegisterRoutes(app *fiber.App, checker *Checker) {
	app.Get("/health", shared.GetLimiter(shared.LimitHealth, nil), checker.Handler)
}
