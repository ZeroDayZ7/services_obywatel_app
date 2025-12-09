package server

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/shared"
)

func SetupGracefulShutdown(app *fiber.App, closeDB func(), timeout time.Duration) {
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-shutdown
		log := shared.GetLogger()
		log.Info("Shutting down server gracefully...")

		if err := app.Shutdown(); err != nil {
			log.Error("Server shutdown failed: " + err.Error())
		}

		if closeDB != nil {
			closeDB()
		}

		log.Info("Server stopped")
		os.Exit(0)
	}()
}
