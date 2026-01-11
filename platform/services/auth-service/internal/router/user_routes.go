package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/auth-service/internal/handler"
)

func SetupUserRoutes(app *fiber.App, h *handler.UserHandler) {
	user := app.Group("/user")
	user.Use(shared.NewLimiter("users", nil))

	user.Get("/sessions", h.GetSessions)
	user.Post("/sessions/terminate", h.TerminateSession)
}
