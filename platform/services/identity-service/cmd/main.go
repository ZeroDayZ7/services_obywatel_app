package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/kms"
	"github.com/zerodayz7/platform/pkg/rabbitmq"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/storage"
	"github.com/zerodayz7/services/identity-service/config"
	"github.com/zerodayz7/services/identity-service/internal/di"
	"github.com/zerodayz7/services/identity-service/internal/router"
	"github.com/zerodayz7/services/identity-service/internal/worker"
)

func LoadSecurityKeys(ctx context.Context, app *config.App, keyStore *httpserver.KeyStore) error {
	log := shared.GetLogger()
	kmsCfg := app.Config.ToKMSServiceConfig()

	log.Info("🔍 Sprawdzanie stanu serwisu KMS...")
	if err := kms.HealthCheck(ctx, kmsCfg); err != nil {
		return err
	}

	// 1. Klucze HTTP
	for senderID, targetKey := range app.Config.HMAC.TargetKeys {
		hmacKey, version, err := kms.FetchSymmetricKeyWithVersion(ctx, kmsCfg, targetKey, "HmacSha256")
		if err != nil {
			return err
		}
		keyStore.SetKey(senderID, hmacKey, version)
		log.Info("✅ Klucz HMAC HTTP załadowany", "service", senderID, "version", version)
	}

	// 2. Klucze RabbitMQ Consumer
	for senderID, targetKey := range app.Config.RabbitConsumers.TrustedSenders {
		hmacKey, version, err := kms.FetchSymmetricKeyWithVersion(ctx, kmsCfg, targetKey, "HmacSha256")
		if err != nil {
			return err
		}
		keyStore.SetKey(senderID, hmacKey, version)
		log.Info("✅ Klucz HMAC Consumer RabbitMQ załadowany", "service", senderID, "version", version)
	}

	// 3. Klucze wewnętrzne serwisu (PESEL, RabbitMQ Publisher, Audyt)
	internalKeys := map[string]string{
		"pesel":    "identity-pesel-blind-index",
		"rabbitmq": "hmac-identity-rabbitmq",
		"audit":    "identity-audit-hmac",
	}

	for keyAlias, kmsTarget := range internalKeys {
		key, version, err := kms.FetchSymmetricKeyWithVersion(ctx, kmsCfg, kmsTarget, "HmacSha256")
		if err != nil {
			return fmt.Errorf("failed to load %s key: %w", keyAlias, err)
		}
		keyStore.SetKey(keyAlias, key, version)
		log.Info("✅ Klucz wewnętrzny załadowany", "alias", keyAlias, "version", version)
	}

	return nil
}

func main() {
	log := shared.GetLogger()

	app, closeDB := config.InitApp()
	defer closeDB()

	keyStore := httpserver.NewKeyStore()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	securityCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := LoadSecurityKeys(securityCtx, app, keyStore); err != nil {
		log.Error("❌ Nie udało się załadować kluczy z KMS", "error", err)
		os.Exit(1)
	}

	// Pobranie klucza do podpisywania zdarzeń wychodzących w RabbitMQ z KeyStore
	rabbitHMACKey, _, ok := keyStore.GetKey("rabbitmq")
	if !ok {
		log.Error("❌ Brak klucza RabbitMQ w KeyStore")
		os.Exit(1)
	}

	var err error
	var eventPublisher rabbitmq.EventPublisher
	if app.Config.RabbitMQ.Enabled {
		log.Info("RabbitMQ is ENABLED. Connecting...")
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

	// =========================================================================
	// INICJALIZACJA S3 STORAGE (NOWY KOD)
	// =========================================================================
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

	// Tworzenie kontenera z przekazaniem fileStorage
	container := di.BuildContainer(app, eventPublisher, app.Config.ToKMSServiceConfig(), keyStore, fileStorage)

	auditWorker := worker.NewAuditWorker(app.DB, eventPublisher, worker.AuditWorkerConfig{
		BatchSize:     app.Config.AuditWorker.BatchSize,
		Interval:      app.Config.AuditWorker.Interval,
		MaxRetries:    app.Config.AuditWorker.MaxRetries,
		BackoffBase:   app.Config.AuditWorker.BackoffBase,
		BackoffMax:    app.Config.AuditWorker.BackoffMax,
		Concurrency:   app.Config.AuditWorker.Concurrency,
		RoutingKey:    app.Config.AuditWorker.RoutingKey,
		SourceService: app.Config.AuditWorker.SourceService,
	})

	log.Info("🚀 Uruchamianie Audit Workera w tle...",
		"batch_size", app.Config.AuditWorker.BatchSize,
		"interval", app.Config.AuditWorker.Interval,
	)

	go auditWorker.Start(ctx)

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
