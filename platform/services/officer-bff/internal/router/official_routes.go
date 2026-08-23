package router

import (
	"net/http"

	"github.com/zerodayz7/services/officer-bff/internal/di"
)

func registerOfficialRoutes(mux *http.ServeMux, c *di.Container) {
	// 1. Istniejące proxy do rejestracji obywatela
	registerProxy, err := NewSingleHostProxy(
		c.Config.IdentityServiceURL,
		"/api/v1/citizens",
		"identity-service",
		c.KeyStore,
	)
	if err != nil {
		panic("failed to create citizen register proxy: " + err.Error())
	}
	mux.HandleFunc("POST /api/v1/official/citizens/register", registerProxy)

	// 2. NOWE PROXY: Przelotowe dla umów (pobieranie/odszyfrowywanie PDF)
	agreementProxy, err := NewSingleHostProxy(
		c.Config.IdentityServiceURL,
		"/api/v1/agreements/",
		"identity-service",
		c.KeyStore,
	)
	if err != nil {
		panic("failed to create agreement download proxy: " + err.Error())
	}

	// Rejestrujemy pod ścieżką, pod którą frontend będzie pytał BFF
	mux.HandleFunc("GET /api/v1/official/agreements/{agreement_id}/download", agreementProxy)
}
