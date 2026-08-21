package router

import (
	"net/http"

	"github.com/zerodayz7/services/officer-bff/internal/di"
)

func registerOfficialRoutes(mux *http.ServeMux, c *di.Container) {
	mux.HandleFunc("POST /api/v1/official/citizens/register", c.OfficialHandler.RegisterCitizen)
}