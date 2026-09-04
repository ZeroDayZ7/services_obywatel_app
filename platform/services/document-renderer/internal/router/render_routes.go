package router

import (
	"document-renderer/config"
	"document-renderer/internal/handler"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(cfg *config.Config, renderHandler handler.RenderHandler, healthHandler handler.HealthHandler) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Serwowanie plików statycznych / assets
	fileServer := http.FileServer(http.Dir(cfg.AssetsDir))
	r.Handle("/assets/*", http.StripPrefix("/assets/", fileServer))

	// Endpoints Health / Probes
	r.Route("/health", func(r chi.Router) {
		r.Get("/", healthHandler.Live)
		r.Get("/live", healthHandler.Live)
		r.Get("/ready", healthHandler.Ready)
	})

	// Sub-router dla v1 API
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/render", renderHandler.RenderPDF)
	})

	return r
}
