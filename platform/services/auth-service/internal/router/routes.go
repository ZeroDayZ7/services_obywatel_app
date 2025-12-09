package router

import (
	authRoutes "github.com/zerodayz7/platform/services/auth-service/internal/features/auth/router"
	userRoutes "github.com/zerodayz7/platform/services/auth-service/internal/features/users/router"

	authHandler "github.com/zerodayz7/platform/services/auth-service/internal/features/auth/handler"
	userHandler "github.com/zerodayz7/platform/services/auth-service/internal/features/users/handler"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(
	app *fiber.App,
	authH *authHandler.AuthHandler,
	userH *userHandler.UserHandler,
) {
	SetupHealthRoutes(app)
	authRoutes.SetupAuthRoutes(app, authH)
	userRoutes.SetupUserRoutes(app, userH)
	SetupFallbackHandlers(app)
}
