package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/auth-service/internal/features/users/handler"
)

func SetupUserRoutes(app *fiber.App, h *handler.UserHandler) {
	// Grupa /user – pasuje do Gatewaya i Fluttera
	user := app.Group("/user")

	// Middleware limitujący requesty
	user.Use(shared.NewLimiter("users", nil))

	// --- SESJE URZĄDZEŃ ---
	// Pobieranie aktywnych sesji (urządzeń)
	user.Get("/sessions", h.GetSessions)

	// Wylogowanie konkretnego urządzenia (terminacja sesji)
	user.Post("/sessions/terminate", h.TerminateSession)

	// --- PROFIL UŻYTKOWNIKA ---
	// protected := user.Group("/profile")
	// protected.Get("/me", h.GetProfile)
	// protected.Patch("/update", h.UpdateProfile)
}
