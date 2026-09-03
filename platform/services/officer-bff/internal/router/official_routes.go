package router

import (
	"net/http"

	"github.com/zerodayz7/services/officer-bff/internal/di"
)

//#region registerOfficialRoutes
func registerOfficialRoutes(mux *http.ServeMux, c *di.Container) {
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

	agreementProxy, err := NewSingleHostProxy(
		c.Config.IdentityServiceURL,
		"/api/v1/agreements/{agreement_id}/download",
		"identity-service",
		c.KeyStore,
	)
	if err != nil {
		panic("failed to create agreement download proxy: " + err.Error())
	}
	mux.HandleFunc("GET /api/v1/official/agreements/{agreement_id}/download", agreementProxy)
}
