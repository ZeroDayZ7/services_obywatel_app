package router

import (
	"net/http"

	"github.com/zerodayz7/platform/pkg/httpserver"
)

func registerHealthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", httpserver.NewHealthHandler())
}