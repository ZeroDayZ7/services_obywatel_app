package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
)

const ServiceOfficerBFF = "officer-bff"

func RegisterOfficerRoutes(app *fiber.App, container *di.Container) {
	target := container.Config.Services.OfficerBFF

	// Grupa tras przeznaczona dla urzędu
	official := app.Group("/api/v1/official")

	// 1. Auth urzędnika (przekierowanie do officer-bff w celu obsługi ciasteczek)
	official.Post("/auth/login", ReverseProxy(container, ServiceOfficerBFF, target))
	official.Post("/auth/logout", ReverseProxy(container, ServiceOfficerBFF, target))

	// 2. Rejestracja obywatela (orkiestrowana w officer-bff)
	official.Post("/citizens/register", ReverseProxy(container, ServiceOfficerBFF, target))
}
