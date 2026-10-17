package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zerodayz7/platform/pkg/httpserver"
	pkgredis "github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/storage" // <-- Import paczki storage
	"github.com/zerodayz7/platform/services/messaging-service/app"
	"github.com/zerodayz7/platform/services/messaging-service/config"
	"github.com/zerodayz7/platform/services/messaging-service/internal/di"
	"github.com/zerodayz7/platform/services/messaging-service/internal/router"
	"github.com/zerodayz7/platform/services/messaging-service/internal/security"
)

func main() {
	// 0. Bootstrap Logger
	bootLog := shared.InitBootstrapLogger(os.Getenv("ENV"), false)
	defer func() { _ = bootLog.Sync() }()

	// 1. Load Config
	if err := config.LoadConfigGlobal(); err != nil {
		bootLog.Fatal("Config load failed", "error", err)
	}

	log := shared.GetLogger()

	// 2. Init KeyStore & Signal Context
	keyStore := httpserver.NewKeyStore()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	securityCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 3. Ładowanie Kluczy z KMS do KeyStore
	rabbitHMACKey, err := security.LoadSecurityKeys(securityCtx, &config.AppConfig, keyStore)
	if err != nil {
		log.Error("❌ Nie udało się załadować kluczy bezpieczeństwa z KMS", "error", err)
		os.Exit(1)
	}
	if config.AppConfig.RabbitMQEnabled && len(rabbitHMACKey) == 0 {
		log.Error("❌ Brak klucza RabbitMQ po załadowaniu z KMS, mimo że RabbitMQ jest włączone")
		os.Exit(1)
	}
	if !config.AppConfig.RabbitMQEnabled && rabbitHMACKey != nil {
		log.Warn("RabbitMQ jest wyłączony, więc zwrócony klucz HMAC powinien być pusty.")
	}

	// 4. Database
	db, closeDB := config.MustInitDB(config.AppConfig.Database)
	defer closeDB()

	// 5. Redis client for websocket event fan-out
	redisClient, err := pkgredis.New(pkgredis.Config(config.AppConfig.Redis))
	if err != nil {
		log.Warn("Redis is not available for messaging-service; websocket notifications will be skipped", "error", err)
	} else {
		defer func() {
			if err := redisClient.Close(); err != nil {
				log.Error("Failed to close Redis client cleanly", "error", err)
			}
		}()
		log.Info("Redis connected for messaging-service websocket notifications", "host", config.AppConfig.Redis.Host, "db", config.AppConfig.Redis.DB)
	}

	// 5.5. S3 / MinIO Storage Initialization (zgodnie z wzorcem z identity-service)
	var fileStorage storage.StorageClient
	if config.AppConfig.S3.Enabled {
		log.Info(
			"S3 Storage is ENABLED. Connecting...",
			"internal_endpoint", config.AppConfig.S3.InternalEndpoint,
			"external_endpoint", config.AppConfig.S3.ExternalEndpoint,
			"bucket", config.AppConfig.S3.Bucket,
			"use_ssl", config.AppConfig.S3.UseSSL,
		)
		s3, err := storage.NewS3Storage(
			config.AppConfig.S3.InternalEndpoint,
			config.AppConfig.S3.ExternalEndpoint,
			config.AppConfig.S3.AccessKey,
			config.AppConfig.S3.SecretKey,
			config.AppConfig.S3.Bucket,
			config.AppConfig.S3.UseSSL,
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

	// 6. DI Container & App Setup (przekazujemy fileStorage)
	container := di.NewContainer(db, redisClient, fileStorage, log, &config.AppConfig, keyStore)
	messagingApp := app.NewApp(container)

	router.SetupMessagingRoutes(messagingApp, container)

	// 7. Run Server
	server.Run(
		messagingApp,
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
