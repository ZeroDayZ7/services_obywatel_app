package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/router"
	"github.com/zerodayz7/platform/pkg/router/health"
	"github.com/zerodayz7/platform/services/auth-service/config"
	"github.com/zerodayz7/platform/services/auth-service/internal/di"
	authRoutes "github.com/zerodayz7/platform/services/auth-service/internal/features/auth/router"
	userRoutes "github.com/zerodayz7/platform/services/auth-service/internal/features/users/router"
)

func SetupRoutes(app *fiber.App, container *di.Container) {
	checker := &health.Checker{
		Redis:     container.Redis.Client,
		Service:   config.AppConfig.Server.AppName,
		Version:   config.AppConfig.Server.AppVersion,
		Upstreams: nil,
	}

	health.RegisterRoutes(app, checker)

	authRoutes.SetupAuthRoutes(app, container.Handlers.AuthHandler, container.Handlers.ResetHandler)
	userRoutes.SetupUserRoutes(app, container.Handlers.UserHandler)

	router.SetupFallbackHandlers(app)
}
