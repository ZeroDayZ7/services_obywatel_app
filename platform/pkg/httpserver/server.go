package httpserver

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zerodayz7/platform/pkg/shared"
)

// Run uruchamia serwer HTTP z wbudowaną obsługą Graceful Shutdown.
func Run(srv *http.Server, shutdownTimeout time.Duration) error {
	log := shared.GetLogger()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Info("HTTP server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("HTTP server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Info("Shutdown signal received, stopping HTTP server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}

	log.Info("HTTP server stopped gracefully")
	return nil
}
