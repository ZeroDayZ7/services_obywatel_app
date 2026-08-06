package main

import (
	"context"
	"os"
	"time"

	"github.com/zerodayz7/platform/pkg/kms"
	"github.com/zerodayz7/platform/pkg/rabbitmq"
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/telemetry"
	"github.com/zerodayz7/platform/services/auth-service/app"
	"github.com/zerodayz7/platform/services/auth-service/config"
	"github.com/zerodayz7/platform/services/auth-service/internal/di"
	"github.com/zerodayz7/platform/services/auth-service/internal/router"
)

func main() {
	// 0. Bootstrap Logger
	bootLog := shared.InitBootstrapLogger(os.Getenv("ENV"), false)
	defer func() { _ = bootLog.Sync() }()

	// 1. Config
	if err := config.LoadConfigGlobal(); err != nil {
		bootLog.Error("Config load failed", "error", err)
		os.Exit(1)
	}

	// 1.1 Bootstrap Keys from KMS (Rust)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	kmsCfg := kms.Config{
		Endpoint:      config.AppConfig.KMS.Endpoint,
		ServiceName:   config.AppConfig.Server.AppName, // "auth-service"
		ServiceSecret: config.AppConfig.KMS.InternalSecret,
	}

	// TUTAJ: Przekazujemy "shared-jwt" jako cel
	privKey, err := kms.FetchAuthPrivateKey(ctx, kmsCfg, "shared-jwt")
	if err != nil {
		bootLog.Error("❌ Krytyczny błąd bootstrapu klucza KMS", "error", err)
		os.Exit(1)
	}

	bootLog.Info("✅ Pomyślnie pobrano i zweryfikowano klucz prywatny z KMS")

	// Wpisujemy pobrany z RAM KMS klucz prywatny do konfiguracji w RAM auth-service
	config.AppConfig.JWT.AccessPrivateKey = privKey

	// 2. Logger
	log := shared.InitLogger(config.AppConfig.Server.Env, false)

	// 3. Telemetry (Tracer)
	if config.AppConfig.OTEL.Enabled {
		cleanup := telemetry.InitTracer(
			config.AppConfig.Server.AppName,
			config.AppConfig.OTEL.Endpoint,
		)
		defer cleanup()
	}

	// 4. Redis
	redisClient, err := redis.New(redis.Config(config.AppConfig.Redis))
	if err != nil {
		log.Error("Redis initialization failed", "error", err)
		os.Exit(1)
	}

	if redisClient == nil {
		log.Error("Redis client is nil")
		os.Exit(1)
	}

	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Error("Failed to close Redis client", "error", err)
		}
	}()

	// 5. Database
	db, closeDB := config.MustInitDB(config.AppConfig.Database)
	defer closeDB()

	// 6. RabbitMQ
	var eventPublisher rabbitmq.EventPublisher

	if config.AppConfig.RabbitMQ.Enabled {
		log.Info("RabbitMQ is ENABLED. Connecting to broker...")
		var err error
		eventPublisher, err = rabbitmq.NewLivePublisher(config.AppConfig.RabbitMQ.GetURL())
		if err != nil {
			log.Error("RabbitMQ initialization failed", "error", err)
			os.Exit(1)
		}
	} else {
		log.Warn("RabbitMQ is DISABLED. Fallback to No-Op Driver.")
		eventPublisher = rabbitmq.NewNoOpPublisher()
	}

	// 7. DI & App Setup
	container := di.NewContainer(db, redisClient, eventPublisher, &config.AppConfig)
	app := app.NewAuthApp(container)

	router.SetupRoutes(app, container)

	// 8. Run server
	server.Run(
		app,
		server.Config{
			Port:       config.AppConfig.Server.Port,
			AppName:    config.AppConfig.Server.AppName,
			AppVersion: config.AppConfig.Server.AppVersion,
			Env:        config.AppConfig.Server.Env,
			Shutdown:   config.AppConfig.Shutdown,
		},
		*log,
		func() {
			log.Info("Shutting down resources")

			closeDB()

			if err := redisClient.Close(); err != nil {
				log.Error("Failed to close Redis client", "error", err)
			}

			if err := eventPublisher.Close(); err != nil {
				log.Error("Failed to close RabbitMQ connection cleanly", "error", err)
			}
		},
	)
}
