package main

import (
	"net/http"
	"time"

	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/services/identity-service/config"
	"github.com/zerodayz7/services/identity-service/internal/di"
	"github.com/zerodayz7/services/identity-service/internal/router"
)

func main() {
	log := shared.GetLogger()

	// 1. Inicjalizacja konfiguracji i bazy danych
	app, closeDB := config.InitApp()
	defer closeDB()

	// 2. Budowanie kontenera zależności (DI)
	container := di.BuildContainer(app)

	// 3. Budowanie routera na natywnym http.ServeMux
	r := router.NewRouter(container)

	// 4. Konfiguracja instancji serwera HTTP
	srv := &http.Server{
		Addr:         ":" + app.Config.Server.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 5. Uruchomienie z Graceful Shutdown przez pkg/httpserver
	if err := httpserver.Run(srv, app.Config.Shutdown); err != nil {
		log.Error("Server forced shutdown with error", "error", err)
	}
}
