package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
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

func symmetricKeyAlgorithmForTarget(target string) string {
	if strings.HasPrefix(target, "hmac-") || strings.Contains(target, "-hmac") || strings.Contains(target, "-index") {
		return "HmacSha256"
	}
	return "AES256GCM"
}

func LoadSecurityKeys(ctx context.Context, app *config.App, keyStore *httpserver.KeyStore) {
	log := shared.GetLogger()
	kmsCfg := app.Config.ToKMSServiceConfig()

	log.Info("🔍 Sprawdzanie stanu serwisu KMS...")
	if err := kms.HealthCheck(ctx, kmsCfg); err != nil {
		log.Error("❌ KMS jest niedostępny podczas inicjalizacji", "error", err)
		os.Exit(1)
	}

	loadKey := func(alias, targetKey string) {
		algorithm := symmetricKeyAlgorithmForTarget(targetKey)
		keyBytes, version, err := kms.FetchSymmetricKeyWithVersion(ctx, kmsCfg, targetKey, 1, algorithm)
		if err != nil {
			log.Error("❌ Nie udało się pobrać klucza z KMS",
				"alias", alias,
				"target", targetKey,
				"algorithm", algorithm,
				"error", err,
			)
			os.Exit(1)
		}

		keyStore.SetKey(alias, keyBytes, uint32(version))
		log.Info("✅ Klucz załadowany do KeyStore",
			"alias", alias,
			"target", targetKey,
			"algorithm", algorithm,
			"version", version,
		)
	}

	for senderID, targetKey := range app.Config.HMAC.TargetKeys {
		loadKey(senderID, targetKey)
	}

	for senderID, targetKey := range app.Config.RabbitConsumers.TrustedSenders {
		loadKey(senderID, targetKey)
	}

	internalKeys := map[string]string{
		"pesel":    "identity-pesel-blind-index",
		"rabbitmq": "hmac-identity-rabbitmq",
		"audit":    "identity-audit-hmac",
	}
	for alias, targetKey := range internalKeys {
		loadKey(alias, targetKey)
	}

	if _, _, ok := keyStore.GetKey("rabbitmq"); !ok {
		log.Error("❌ Brak klucza RabbitMQ w KeyStore po załadowaniu z KMS")
		os.Exit(1)
	}
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

	LoadSecurityKeys(securityCtx, app, keyStore)

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
	// INICJALIZACJA S3 STORAGE
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
