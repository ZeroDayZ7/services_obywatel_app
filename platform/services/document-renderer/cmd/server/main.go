package main

import (
	"document-renderer/config"
	"document-renderer/internal/di"
	"document-renderer/internal/renderer"
	"log"
	"net/http"
	"time"

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

	pdfRenderer := renderer.NewRodPDFRenderer(browser, cfg.PDFMaxConcurrency, cfg.PDFRenderTimeout)
	container := di.NewContainer(&cfg, pdfRenderer)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      container.Router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	shutdownTimeout := 10 * time.Second

	if err := httpserver.Run(server, shutdownTimeout); err != nil {
		log.Fatalf("Server forced to shutdown with error: %v", err)
	}
}
