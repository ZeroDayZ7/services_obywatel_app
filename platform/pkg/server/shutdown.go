package server

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/zerodayz7/platform/pkg/shared"
)

// SetupGracefulShutdown nasłuchuje sygnałów zakończenia i bezpiecznie zamyka aplikację.
// Przyjmuje instancję Fiber, timeout oraz dowolną liczbę funkcji sprzątających (cleanups).
func SetupGracefulShutdown(app *fiber.App, timeout time.Duration, cleanups ...func()) {
	shutdown := make(chan os.Signal, 1)
	// SIGINT (Ctrl+C), SIGTERM (Docker/K8s stop)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-shutdown
		log := shared.GetLogger()
		log.Info("Otrzymano sygnał zamknięcia. Rozpoczynanie graceful shutdown...")

		// Tworzymy timer na wypadek, gdyby zamykanie trwało zbyt długo
		// Fiber.ShutdownWithTimeout(timeout) jest dostępny w nowszych wersjach,
		// ale zrobimy to uniwersalnie:

		// 1. Zamykamy serwer HTTP (przestaje przyjmować nowe połączenia)
		if err := app.ShutdownWithTimeout(timeout); err != nil {
			log.ErrorObj("Błąd podczas zamykania serwera Fiber", err)
		}

		// 2. Wykonujemy wszystkie przekazane funkcje czyszczące (DB, Redis, itp.)
		for _, cleanup := range cleanups {
			if cleanup != nil {
				cleanup()
			}
		}

		log.Info("Wszystkie zasoby zostały zwolnione. Serwis zatrzymany.")
		os.Exit(0)
	}()
}
