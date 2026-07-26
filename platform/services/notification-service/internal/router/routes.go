package router

import (
	"github.com/gofiber/fiber/v2"
	pkgRouter "github.com/zerodayz7/platform/pkg/router"
	"github.com/zerodayz7/platform/pkg/router/health"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/notification-service/internal/di"
)

func SetupRoutes(app *fiber.App, container *di.Container) {
	h := container.Handlers.NotificationHandler

	checker := &health.Checker{
		Service: "notification-service",
		Version: container.Config.Server.AppVersion,
	}
	health.RegisterRoutes(app, checker)

	notifications := app.Group("/notifications")
	{

		notifications.Use(shared.GetLimiter(shared.LimitNotifications, nil))

		notifications.Get("/", h.ListMyNotifications)
		notifications.Post("/send", h.SendNotification)
		notifications.Patch("/:id/read", h.MarkAsRead)
		notifications.Patch("/read-all", h.MarkAllAsRead)
		notifications.Patch("/:id/trash", h.MoveToTrash)
		notifications.Delete("/trash", h.ClearTrash)
		notifications.Patch("/:id/restore", h.RestoreFromTrash)
		notifications.Delete("/:id", h.DeletePermanently)
	}

	pkgRouter.SetupFallbackHandlers(app)
}
