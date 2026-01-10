package router

import (
	"github.com/gofiber/fiber/v2"
	pkgRouter "github.com/zerodayz7/platform/pkg/router"
	"github.com/zerodayz7/platform/pkg/router/health"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/audit-service/internal/di"
)

// SetupRoutes konfiguruje trasy dla serwisu audytu (przeznaczone głównie dla administracji).
func SetupRoutes(app *fiber.App, container *di.Container) {
	// 1. Inicjalizacja handlera z kontenera DI.
	h := container.AuditHandler

	// 2. Automatyczne Health Checks (spójne z resztą platformy).
	checker := &health.Checker{
		Service: "audit-service",
		Version: container.Config.Server.AppVersion,
	}
	health.RegisterRoutes(app, checker)

	// 3. Grupa audit z dedykowanym limiterem.
	auditGroup := app.Group("/audit")
	{
		// Nałożenie limitera z pkg/shared.
		auditGroup.Use(shared.NewLimiter("audit"))

		// --- Odczyt logów ---
		// Pobieranie listy wszystkich logów (z paginacją).
		auditGroup.Get("/", h.ListLogs)

		// Pobieranie logów konkretnego użytkownika.
		auditGroup.Get("/user/:id", h.ListUserLogs)

		// Pobieranie szczegółów jednego logu (zaszyfrowane dane binarne).
		auditGroup.Get("/:id", h.GetLogDetails)

		// Filtrowanie po typie akcji (np. "login", "document_download").
		auditGroup.Get("/action/:action", h.ListLogsByAction)
	}

	// 4. Uniwersalne Fallback Handlery (404, favicon itp.) z pkg.
	pkgRouter.SetupFallbackHandlers(app)
}
