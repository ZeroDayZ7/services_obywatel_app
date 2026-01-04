package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/audit-service/internal/audit"
)

// SetupAuditRoutes konfiguruje trasy dla zarządzania logami (głównie dla Admina)
func SetupAuditRoutes(app *fiber.App, auditH *audit.AuditHandler) {
	// Tworzymy grupę audit
	auditGroup := app.Group("/audit")

	// Nakładamy limiter z Twojego pkg/shared
	auditGroup.Use(shared.NewLimiter("audit"))

	// --- Odczyt logów (Wymagany klucz publiczny admina do deszyfrowania po stronie klienta) ---

	// Pobieranie listy wszystkich logów (z paginacją)
	// GET /audit
	auditGroup.Get("/", auditH.ListLogs)

	// Pobieranie logów konkretnego użytkownika
	// GET /audit/user/:id
	auditGroup.Get("/user/:id", auditH.ListUserLogs)

	// Pobieranie szczegółów jednego logu (zwraca binarne zaszyfrowane pola)
	// GET /audit/:id
	auditGroup.Get("/:id", auditH.GetLogDetails)

	// Filtrowanie po akcji (np. "login", "payment_failed")
	// GET /audit/action/:action
	auditGroup.Get("/action/:action", auditH.ListLogsByAction)
}
