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

	log := shared.InitLogger(config.AppConfig.Server.Env, false)

	// =========================================================================
	// 2. KMS SETUP & BOOTSTRAP KEYS (JWT Private Key + Internal HMAC)
	// =========================================================================
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Dedykowane konfiguracje KMS
	kmsServiceCfg := config.AppConfig.ToKMSServiceConfig()

	log.Info("🔍 Sprawdzanie stanu serwisu KMS...")
	if err := kms.HealthCheck(ctx, kmsServiceCfg); err != nil {
		log.Error("❌ KMS Health Check nie powiódł się", "error", err)
		os.Exit(1)
	}

	// 2a. Pobranie klucza prywatnego Ed25519 do podpisywania Access Tokenów
	if config.AppConfig.JWT.SigningMode == "local" {
		log.Info("🔑 [MODE: LOCAL] Pobieranie klucza prywatnego JWT z KMS do pamięci serwisu...")
		privKey, err := kms.FetchAuthPrivateKey(ctx, kmsServiceCfg, "shared-jwt")
		if err != nil {
			log.Error("❌ Krytyczny błąd pobierania klucza prywatnego JWT z KMS", "error", err)
			os.Exit(1)
		}
		config.AppConfig.JWT.AccessPrivateKey = privKey
		log.Info("✅ Pomyślnie pobrano i załadowano klucz prywatny JWT do pamięci")
	} else {
		log.Info("🛡️ [MODE: KMS] Tokeny będą podpisywane zdalnie przez API KMS")
	}

	// 2b. Pobranie klucza HMAC do weryfikacji/podpisywania komunikacji między-serwisowej
	log.Info("🔑 Pobieranie klucza 'hmac-gateway-auth' z KMS...")
	internalHMACKey, err := kms.FetchSymmetricKey(ctx, kmsServiceCfg, "hmac-gateway-auth", "HmacSha256")
	if err != nil {
		log.Error("❌ Nie udało się pobrać klucza HMAC z KMS", "error", err)
		os.Exit(1)
	}
	config.AppConfig.Internal.HMACSecret = string(internalHMACKey)
	log.Info("✅ Klucz HMAC komunikacji wewnętrznej pobrany pomyślnie")

	// 4. Telemetry (Tracer)
	if config.AppConfig.OTEL.Enabled {
		cleanup := telemetry.InitTracer(
			config.AppConfig.Server.AppName,
			config.AppConfig.OTEL.Endpoint,
		)
		defer cleanup()
	}

	// 5. Redis
	redisClient, err := redis.New(redis.Config(config.AppConfig.Redis))
	if err != nil || redisClient == nil {
		log.Error("Redis initialization failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Error("Failed to close Redis client", "error", err)
		}
	}()

	// 6. Database
	db, closeDB := config.MustInitDB(config.AppConfig.Database)
	defer closeDB()

	// 7. RabbitMQ
	var eventPublisher rabbitmq.EventPublisher
	if config.AppConfig.RabbitMQ.Enabled {
		log.Info("RabbitMQ is ENABLED. Connecting to broker...")
		eventPublisher, err = rabbitmq.NewLivePublisher(config.AppConfig.RabbitMQ.GetURL())
		if err != nil {
			log.Error("RabbitMQ initialization failed", "error", err)
			os.Exit(1)
		}
	} else {
		log.Warn("RabbitMQ is DISABLED. Fallback to No-Op Driver.")
		eventPublisher = rabbitmq.NewNoOpPublisher()
	}
	defer func() {
		if err := eventPublisher.Close(); err != nil {
			log.Error("Failed to close RabbitMQ connection cleanly", "error", err)
		}
	}()

	// 8. DI Container & App Setup
	container := di.NewContainer(db, redisClient, eventPublisher, &config.AppConfig)
	authApp := app.NewAuthApp(container)

	router.SetupRoutes(authApp, container)

	// 9. Run server
	server.Run(
		authApp,
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
