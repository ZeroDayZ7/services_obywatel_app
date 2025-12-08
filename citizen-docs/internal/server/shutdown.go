package server

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/http-server/internal/shared/logger"
	"go.uber.org/zap"
)

func SetupGracefulShutdown(app *fiber.App, closeDB func(), timeout time.Duration) {
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-shutdown
		logger.GetLogger().Info("Shutting down server gracefully...")

		if err := app.Shutdown(); err != nil {
			logger.GetLogger().Error("Server shutdown failed", zap.Error(err))
		}

		if closeDB != nil {
			closeDB()
		}

		logger.GetLogger().Info("Server stopped")
		os.Exit(0)
	}()
}
