package router

import (
	"github.com/gofiber/fiber/v2"
	handler "github.com/zerodayz7/platform/services/auth-service/internal/handler"
)

func SetupWellKnownRoutes(app *fiber.App, h *handler.WellKnownHandler) {
	wellKnown := app.Group("/.well-known")
	wellKnown.Get("/jwks.json", h.GetJWKS)
}
