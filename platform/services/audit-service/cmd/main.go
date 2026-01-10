package main

import (
	"fmt"

	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/audit-service/config"
	"github.com/zerodayz7/platform/services/audit-service/internal/di"
	"github.com/zerodayz7/platform/services/audit-service/internal/router"
)

func main() {
	// 1. Ładowanie konfiguracji globalnej z pkg/viper.
	if err := config.LoadConfigGlobal(); err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	// 2. Inicjalizacja loggera dla środowiska zdefiniowanego w configu.
	log := shared.InitLogger(config.AppConfig.Server.Env)

	// 3. Nawiązanie połączenia z bazą danych i rejestracja funkcji zamykającej.
	db, closeDB := config.MustInitDB(config.AppConfig.Database)
	defer closeDB()

	// 4. Budowa kontenera zależności DI.
	container := di.NewContainer(db, nil, log, &config.AppConfig)

	// 5. Instancja aplikacji Fiber z wstrzykniętym kontenerem.
	app := config.NewAuditApp(container)

	// 6. Konfiguracja tras routingu przy użyciu ustandaryzowanego SetupRoutes.
	router.SetupRoutes(app, container)

	// 7. Obsługa bezpiecznego wyłączania serwera (poprawiona kolejność argumentów: timeout, cleanups).
	server.SetupGracefulShutdown(
		app,
		config.AppConfig.Shutdown, // time.Duration
		closeDB,
	)

	// 8. Uruchomienie nasłuchiwania serwera z rozszerzonym logowaniem wersji.
	address := ":" + config.AppConfig.Server.Port
	log.Info("Service started", map[string]any{
		"app":     config.AppConfig.Server.AppName,
		"version": config.AppConfig.Server.AppVersion,
		"address": address,
		"env":     config.AppConfig.Server.Env,
	})

	if err := app.Listen(address); err != nil {
		log.ErrorObj("Critical server failure", err)
	}
}
