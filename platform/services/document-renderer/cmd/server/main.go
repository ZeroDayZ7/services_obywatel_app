package main

import (
	"document-renderer/config"
	"document-renderer/internal/di"
	"document-renderer/internal/renderer"
	"net/http"
	"os"

	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/shared"
)

func main() {
	bootLog := shared.InitBootstrapLogger(os.Getenv("ENV"), false)

	cfg, err := config.Load()
	if err != nil {
		bootLog.Error("Configuration error", err)
		os.Exit(1)
	}

	logger := shared.InitLogger(cfg.Env, false)

	logger.Info("Launching Headless Chromium...", "bin", cfg.ChromeBin)
	browser, err := renderer.NewBrowser(cfg.ChromeBin)
	if err != nil {
		logger.Fatal("Failed to launch browser", err)
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

	logger.Info("Starting HTTP server", "env", cfg.Env, "port", cfg.Port)

	if err := httpserver.Run(server, cfg.ShutdownTimeout); err != nil {
		logger.Fatal("Server forced to shutdown with error", err)
	}
}
