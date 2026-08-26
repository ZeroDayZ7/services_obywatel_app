package router

import (
	"net/http"

	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/services/identity-service/internal/di"
)

// NewRouter tworzy i konfiguruje główny multiplexer HTTP usługi.
func NewRouter(container *di.Container) http.Handler {
	mux := http.NewServeMux()

	registerHealthRoutes(mux, container)
	registerCitizenRoutes(mux, container)

	return SetupMiddleware(mux, container)
}

func registerHealthRoutes(mux *http.ServeMux, container *di.Container) {
	mux.HandleFunc("GET /health", httpserver.NewHealthHandler(container.DB.Ping))
}

func registerCitizenRoutes(mux *http.ServeMux, container *di.Container) {
	handler := container.CitizenHandler

	mux.HandleFunc("POST /api/v1/citizens", handler.Register)
	mux.HandleFunc("GET /api/v1/citizens/{user_id}", handler.GetByID)
	mux.HandleFunc("GET /api/v1/agreements/{agreement_id}/download", handler.DownloadAgreementPDF)
}
