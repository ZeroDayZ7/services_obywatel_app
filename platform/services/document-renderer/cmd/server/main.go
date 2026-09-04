package main

import (
	"context"
	"document-renderer/config"
	"document-renderer/internal/renderer"
	"document-renderer/internal/templates"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	api "document-renderer/internal/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	cfg := config.Load()

	log.Println("Launching Headless Chromium...")
	browser, err := renderer.NewBrowser(cfg.ChromeBin)
	if err != nil {
		log.Fatalf("failed to launch browser: %v", err)
	}
	defer browser.Close()

	templateLoader := templates.NewLoader(cfg.TemplatesDir)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Serwowanie zasobów statycznych (CSS, logo, czcionki) dla szablonów
	fileServer := http.FileServer(http.Dir(cfg.AssetsDir))
	r.Handle("/assets/*", http.StripPrefix("/assets/", fileServer))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Generic PDF Generation Endpoint
	r.Post("/api/v1/render", api.HandleRenderDocument(browser, templateLoader))

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	go func() {
		log.Printf("Document Renderer listening on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
