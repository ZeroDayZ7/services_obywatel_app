package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/shared"
)

// --- (FIBER) ---

type Config struct {
	Port       string
	AppName    string
	AppVersion string
	Env        string
	Shutdown   time.Duration
}

func Run(app *fiber.App, cfg Config, log shared.Logger, cleanup func()) {
	SetupGracefulShutdown(app, cfg.Shutdown, cleanup)

	address := fmt.Sprintf("0.0.0.0:%s", cfg.Port)

	log.Info("Service started", map[string]any{
		"app":     cfg.AppName,
		"version": cfg.AppVersion,
		"address": address,
		"env":     cfg.Env,
	})

	if err := app.Listen(address); err != nil {
		log.ErrorObj("Failed to start server", err)
	}
}

// - (NET/HTTP) -
// - VERSION SERVICE -

type Server struct {
	httpServer *http.Server
}

func New(handler http.Handler) *Server {
	return &Server{
		httpServer: &http.Server{
			Handler: handler,
		},
	}
}

func (s *Server) Start(addr string) error {
	s.httpServer.Addr = addr
	return s.httpServer.ListenAndServe()
}
