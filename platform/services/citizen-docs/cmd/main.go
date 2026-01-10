package main

import (
	"fmt"

	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/citizen-docs/config"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/di"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/router"
)

func main() {
	// 1. Wczytanie konfiguracji (używamy Twojej nowej metody LoadConfigGlobal)
	if err := config.LoadConfigGlobal(); err != nil {
		// Używamy fmt.Printf, bo logger jeszcze nie jest zainicjowany poprawnie z configu
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	// 2. Inicjalizacja Loggera (pobieramy środowisko z AppConfig)
	log := shared.InitLogger(config.AppConfig.Server.Env)

	// 3. Inicjalizacja bazy danych (z automatyczną migracją nowych modeli)
	db, closeDB := config.MustInitDB(config.AppConfig.Database)
	defer closeDB()

	// 4. Inicjalizacja DI Container (przekazujemy db, log i wskaźnik na config)
	// Upewnij się, że Twoje di.NewContainer przyjmuje (db, log, config)
	container := di.NewContainer(db, log, &config.AppConfig)

	// 5. Tworzenie instancji Fiber App (przekazujemy cały kontener DI)
	app := config.NewDocsApp(container)

	// 6. Konfiguracja routingu (używamy handlerów z kontenera)
	router.SetupDocsRoutes(app, container.UserDocumentSvc)

	// 7. Pancerne Graceful Shutdown (POPRAWIONA KOLEJNOŚĆ: app, timeout, cleanups)
	server.SetupGracefulShutdown(
		app,
		config.AppConfig.Shutdown, // 2nd: time.Duration
		closeDB,                   // 3rd: variadic func()
	)

	// 8. Uruchomienie serwera
	address := ":" + config.AppConfig.Server.Port
	log.Info("Citizen Docs Service started", map[string]any{
		"app":     config.AppConfig.Server.AppName,
		"version": config.AppConfig.Server.AppVersion,
		"address": address,
	})

	if err := app.Listen(address); err != nil {
		log.ErrorObj("Critical server failure", err)
	}
}
