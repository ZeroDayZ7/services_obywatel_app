package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/utils"
	"github.com/zerodayz7/platform/services/audit-service/app"
	"github.com/zerodayz7/platform/services/audit-service/config"
	"github.com/zerodayz7/platform/services/audit-service/internal/di"
	"github.com/zerodayz7/platform/services/audit-service/internal/router"
)

func main() {
	bootLog := shared.InitBootstrapLogger(os.Getenv("ENV"), false)
	defer func() { _ = bootLog.Sync() }()

	if err := config.LoadConfigGlobal(); err != nil {
		bootLog.Fatal("Config load failed", "error", err)
	}

	log := shared.InitLogger(config.AppConfig.Server.Env, false)

	// 1. Baza danych
	db, closeDB := config.MustInitDB(config.AppConfig.Database)
	defer closeDB()

	// 2. Inicjalizacja Redis z wykorzystaniem pkg/redis
	rdb, err := redis.New(redis.Config(config.AppConfig.Redis))
	if err != nil {
		log.Fatal("Redis initialization failed", "error", err)
	}
	defer func() {
		if err := rdb.Close(); err != nil {
			log.ErrorObj("Failed to close Redis connection", err)
		}
	}()

	// 3. DI Container z poprawną instancją Redisa
	container := di.NewContainer(db, rdb, log, &config.AppConfig)

	// 4. Kontekst dla obsługi wyłączenia aplikacji (Graceful Shutdown)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 5. Uruchomienie workera
	utils.SafeGo(log, func() {
		container.AuditWorker.Start(ctx)
	})

	// 6. Router i Serwer Fiber
	auditApp := app.NewAuditApp(container)
	router.SetupRoutes(auditApp, container)

	server.Run(
		auditApp,
		server.Config{
			Port:       config.AppConfig.Server.Port,
			AppName:    config.AppConfig.Server.AppName,
			AppVersion: config.AppConfig.Server.AppVersion,
			Env:        config.AppConfig.Server.Env,
			Shutdown:   config.AppConfig.Shutdown,
		},
		*log,
		func() {
			stop() // Wysłanie sygnału zakreślenia kontekstu do workera
		},
	)
}
