package router

import (
	"net/http"

	"github.com/zerodayz7/services/officer-bff/internal/di"
)

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

	mux.HandleFunc("GET /api/v1/official/agreements/{agreement_id}/download", func(w http.ResponseWriter, r *http.Request) {
		agreementID := r.PathValue("agreement_id")

		agreementProxy, err := NewSingleHostProxy(
			c.Config.IdentityServiceURL,
			"/api/v1/agreements/"+agreementID+"/download",
			"identity-service",
			c.KeyStore,
		)
		if err != nil {
			http.Error(w, "failed to create agreement download proxy", http.StatusInternalServerError)
			return
		}

		agreementProxy(w, r)
	})
}
