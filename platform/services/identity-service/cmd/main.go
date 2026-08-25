package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/rabbitmq"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/storage"
	"github.com/zerodayz7/services/identity-service/config"
	"github.com/zerodayz7/services/identity-service/internal/di"
	"github.com/zerodayz7/services/identity-service/internal/router"
	"github.com/zerodayz7/services/identity-service/internal/security"
	"github.com/zerodayz7/services/identity-service/internal/worker"
)

//#region main
func main() {
	log := shared.GetLogger()

	app, closeDB := config.InitApp()
	defer closeDB()

	keyStore := httpserver.NewKeyStore()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	securityCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Załaduj klucze bezpieczeństwa z KMS do KeyStore
	if err := security.LoadSecurityKeys(securityCtx, app, keyStore); err != nil {
		log.Error("❌ Błąd ładowania kluczy bezpieczeństwa", "error", err)
		os.Exit(1)
	}

	var eventPublisher rabbitmq.EventPublisher
	var err error

	if app.Config.RabbitMQ.Enabled {
		log.Info("RabbitMQ is ENABLED. Connecting...")

		rabbitHMACKey, _, ok := keyStore.GetKey("rabbitmq")
		if !ok {
			log.Error("❌ Brak klucza RabbitMQ w KeyStore")
			os.Exit(1)
		}

		eventPublisher, err = rabbitmq.NewLivePublisher(
			app.Config.RabbitMQ.GetURL(),
			app.Config.Server.AppName,
			rabbitHMACKey,
		)
		if err != nil {
			log.Error("❌ Nie udało się połączyć z RabbitMQ", "error", err)
			os.Exit(1)
		}
	} else {
		log.Warn("RabbitMQ is DISABLED. Fallback to No-Op Driver.")
		eventPublisher = rabbitmq.NewNoOpPublisher()
	}

	defer func() {
		if err := eventPublisher.Close(); err != nil {
			log.Error("Błąd podczas zamykania połączenia z RabbitMQ", "error", err)
		}
	}()

	var fileStorage storage.StorageClient
	if app.Config.S3.Enabled {
		log.Info("S3 Storage is ENABLED. Connecting...")
		s3, err := storage.NewS3Storage(
			app.Config.S3.Endpoint,
			app.Config.S3.AccessKey,
			app.Config.S3.SecretKey,
			app.Config.S3.Bucket,
			app.Config.S3.UseSSL,
		)
		if err != nil {
			log.Error("❌ S3 initialization failed", "error", err)
			os.Exit(1)
		}
		fileStorage = s3
	} else {
		log.Warn("S3 Storage is DISABLED. Using No-Op storage.")
		fileStorage = &storage.NoOpStorage{}
	}

	container := di.BuildContainer(app, eventPublisher, app.Config.ToKMSServiceConfig(), keyStore, fileStorage)

	auditWorker := worker.NewAuditWorker(
		app.DB,
		eventPublisher,
		app.Config.AuditWorker.ToWorkerConfig(),
	)

	if app.Config.AuditWorker.Enabled {
		log.Info("🚀 Uruchamianie Audit Workera w tle...",
			"batch_size", app.Config.AuditWorker.BatchSize,
			"interval", app.Config.AuditWorker.Interval,
		)
		go auditWorker.Start(ctx)
	} else {
		log.Warn("⚠️ Audit Worker jest wyłączony (Enabled=false).")
	}

	registrationWorker := worker.NewRegistrationWorker(
		app.DB,
		eventPublisher,
		app.Config.RegistrationWorker.ToWorkerConfig(),
	)

	if app.Config.RegistrationWorker.Enabled {
		log.Info("🚀 Uruchamianie Registration Workera w tle...",
			"batch_size", app.Config.RegistrationWorker.BatchSize,
			"interval", app.Config.RegistrationWorker.Interval,
		)
		go registrationWorker.Start(ctx)
	} else {
		log.Warn("⚠️ Registration Worker jest wyłączony (Enabled=false).")
	}

	r := router.NewRouter(container)
	server := &http.Server{
		Addr:         ":" + app.Config.Server.Port,
		Handler:      r,
		ReadTimeout:  app.Config.Server.ReadTimeout,
		WriteTimeout: app.Config.Server.WriteTimeout,
		IdleTimeout:  app.Config.Server.IdleTimeout,
	}

	if err := httpserver.Run(server, app.Config.Shutdown); err != nil {
		log.Error("Server forced shutdown with error", "error", err)
	}
}
