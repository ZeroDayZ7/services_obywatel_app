package router

import (
	"net/http"

	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/services/officer-bff/internal/di"
)

func NewRouter(c *di.Container) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", httpserver.NewHealthHandler())
	mux.HandleFunc("POST /api/v1/official/citizens/register", c.OfficialHandler.RegisterCitizen)

	return mux
}
