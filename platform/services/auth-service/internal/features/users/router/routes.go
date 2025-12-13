package router

import (
	"github.com/gofiber/fiber/v2"
	// "github.com/gofiber/fiber/v2/middleware/csrf"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/auth-service/internal/features/users/handler"
)

func SetupUserRoutes(app *fiber.App, h *handler.UserHandler) {
	users := app.Group("/users")
	protected := users.Group("/")
	protected.Use(shared.NewLimiter("users"))

	// Test sesji – handler pobiera sesję z c.Locals("session")

	// Middleware CSRF dla wybranych endpointów
	// csrfProtected := protected.Group("/")
	// csrfProtected.Use(csrf.New(config.NewCSRFConfig(config.SessionStore().Storage)))

	// Test CSRF – handler pobiera token z c.Locals("csrf")
	// csrfProtected.Post("/test-csrf", h.TestCSRF)

	// Przykładowe routy użytkownika
	// protected.Get("/me", h.GetProfile)
	// protected.Post("/update", h.UpdateProfile)
}
