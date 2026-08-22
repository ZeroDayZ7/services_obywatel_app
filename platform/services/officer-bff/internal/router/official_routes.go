package router

import (
	"net/http"

	"github.com/zerodayz7/services/officer-bff/internal/di"
)

func registerOfficialRoutes(mux *http.ServeMux, c *di.Container) {
	// Rejestracja proxy przelotowego do identity-service
	registerProxy, err := NewSingleHostProxy(
		c.Config.IdentityServiceURL, // URL identity-service z konfiguracji
		"/api/v1/citizens",          // Ścieżka docelowa w identity-service
		"identity-service",          // ID serwisu targetowego (dla podpisu HMAC)
		c.KeyStore,                  // Magazyn kluczy do HMAC
	)
	if err != nil {
		panic("failed to create citizen register proxy: " + err.Error())
	}

	mux.HandleFunc("POST /api/v1/official/citizens/register", registerProxy)
}
