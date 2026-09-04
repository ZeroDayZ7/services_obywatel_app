package main

import (
	"document-renderer/config"
	"document-renderer/internal/di"
	"document-renderer/internal/renderer"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/zerodayz7/platform/pkg/httpserver"
)

func main() {
	cfg := config.Load()

	log.Println("Launching Headless Chromium...")
	browser, err := renderer.NewBrowser(cfg.ChromeBin)
	if err != nil {
		log.Fatalf("failed to launch browser: %v", err)
	}
	defer browser.Close()

	pdfRenderer := renderer.NewRodPDFRenderer(browser)
	container := di.NewContainer(&cfg, pdfRenderer)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	fileServer := http.FileServer(http.Dir(cfg.AssetsDir))
	r.Handle("/assets/*", http.StripPrefix("/assets/", fileServer))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	r.Mount("/api/v1", container.Router)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	shutdownTimeout := 10 * time.Second

	if err := httpserver.Run(server, shutdownTimeout); err != nil {
		log.Fatalf("Server forced to shutdown with error: %v", err)
	}
}
