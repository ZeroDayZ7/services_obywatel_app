package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zerodayz7/platform/pkg/envelope"
	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/citizen-docs/app"
	"github.com/zerodayz7/platform/services/citizen-docs/config"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/di"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/router"
)

func main() {
	// 1. Ładowanie Konfiguracji
	if err := config.LoadConfigGlobal(); err != nil {
		shared.GetLogger().Error("Config load failed", "error", err)
		os.Exit(1)
	}

	log := shared.GetLogger()

	// 2. Inicjalizacja KeyStore i Kontekstu
	keyStore := httpserver.NewKeyStore()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	securityCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 3. Ładowanie Kluczy z KMS
	LoadSecurityKeys(securityCtx, &config.AppConfig, keyStore)

	// 4. Inicjalizacja Cryptor, Baza Danych i DI Container
	cryptor := envelope.NewEnvelopeCryptor(config.AppConfig.ToKMSServiceConfig())

	db, closeDB := config.MustInitDB(config.AppConfig.Database)
	defer closeDB()

	container := di.NewContainer(db, log, &config.AppConfig, cryptor)

	// 5. Aplikacja i Router
	docsApp := app.NewDocsApp(container)
	router.SetupDocsRoutes(docsApp, container)

	// 6. Start Serwera
	server.Run(
		docsApp,
		server.Config{
			Port:       config.AppConfig.Server.Port,
			AppName:    config.AppConfig.Server.AppName,
			AppVersion: config.AppConfig.Server.AppVersion,
			Env:        config.AppConfig.Server.Env,
			Shutdown:   config.AppConfig.Shutdown,
		},
		*log,
		nil,
	)
}
