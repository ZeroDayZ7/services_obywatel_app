package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/shared"
	"golang.org/x/sync/errgroup"
)

type Config struct {
	Port       string
	AppName    string
	AppVersion string
	Env        string
	Shutdown   time.Duration
}

func Run(app *fiber.App, cfg Config, log shared.Logger, cleanup func()) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	g, gCtx := errgroup.WithContext(ctx)

	address := fmt.Sprintf("0.0.0.0:%s", cfg.Port)

	g.Go(func() error {
		log.Info("Service started", map[string]any{
			"app":     cfg.AppName,
			"version": cfg.AppVersion,
			"address": address,
			"env":     cfg.Env,
		})

		return app.Listen(address)
	})

	g.Go(func() error {
		<-gCtx.Done()

		log.Info("Shutdown signal received. Starting graceful shutdown...", nil)

		err := app.ShutdownWithTimeout(cfg.Shutdown)
		if err != nil {
			log.ErrorObj("Fiber shutdown failed", err)
		}

		if cleanup != nil {
			cleanup()
			log.Info("Infrastructure resources shut down successfully", nil)
		}

		return err
	})

	if err := g.Wait(); err != nil {
		log.ErrorObj("Application exited with error", err)
		os.Exit(1)
	}

	log.Info("Service stopped cleanly", nil)
}
