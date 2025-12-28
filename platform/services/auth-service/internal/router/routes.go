package router

import (
	authRoutes "github.com/zerodayz7/platform/services/auth-service/internal/features/auth/router"
	userRoutes "github.com/zerodayz7/platform/services/auth-service/internal/features/users/router"

	authHandler "github.com/zerodayz7/platform/services/auth-service/internal/features/auth/handler"
	resetHandler "github.com/zerodayz7/platform/services/auth-service/internal/features/auth/handler"
	userHandler "github.com/zerodayz7/platform/services/auth-service/internal/features/users/handler"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(
	app *fiber.App,
	authH *authHandler.AuthHandler,
	resetH *resetHandler.ResetHandler,
	userH *userHandler.UserHandler,
) {
	SetupHealthRoutes(app)

	// auth z reset handlerem
	authRoutes.SetupAuthRoutes(app, authH, resetH)

	userRoutes.SetupUserRoutes(app, userH)
	SetupFallbackHandlers(app)
}
