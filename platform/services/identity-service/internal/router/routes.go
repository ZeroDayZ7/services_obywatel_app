package router

import (
	"net/http"

	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/services/identity-service/internal/di"
)

func NewRouter(c *di.Container) http.Handler {
	mux := http.NewServeMux()

	// Health check weryfikujący połączenie z bazą przez c.DB.Ping
	mux.HandleFunc("GET /health", httpserver.NewHealthHandler(c.DB.Ping))

	// Citizen API
	mux.HandleFunc("POST /api/v1/citizens", c.CitizenHandler.Register)
	mux.HandleFunc("GET /api/v1/citizens/{user_id}", c.CitizenHandler.GetByID)

	return applyGlobalMiddleware(mux, c)
}

func applyGlobalMiddleware(handler http.Handler, c *di.Container) http.Handler {
	log := shared.GetLogger()

	// Pobieramy klucz HMAC przypisany do usugi officer-bff
	bffSecret, _, ok := c.KeyStore.GetKey("officer-bff")
	if !ok {
		log.Warn("⚠️ Brak klucza HMAC dla officer-bff w KeyStore identity-service!")
	}

	hmacMiddleware := httpserver.InternalAuthMiddleware(bffSecret)
	loggerMiddleware := httpserver.LoggerMiddleware(log)

	handler = hmacMiddleware(handler)
	handler = loggerMiddleware(handler)

	return handler
}
