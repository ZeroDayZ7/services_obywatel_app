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
	"github.com/zerodayz7/platform/services/gateway/app"
	"github.com/zerodayz7/platform/services/gateway/config"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
	"github.com/zerodayz7/platform/services/gateway/internal/hmac"
	"github.com/zerodayz7/platform/services/gateway/internal/router"
)

// loadSecurityKeys obsługuje połączenie z KMS, health-check oraz pobieranie kluczy JWT, HMAC serwisów i opcjonalnie RabbitMQ.
func loadSecurityKeys(ctx context.Context, appCfg *config.Config, keyStore *hmac.GatewayKeyStore) []byte {
	log := shared.GetLogger()
	kmsCfg := appCfg.ToKMSServiceConfig()

	log.Info("🔍 Sprawdzanie stanu serwisu KMS...")
	if err := kms.HealthCheck(ctx, kmsCfg); err != nil {
		log.Error("❌ KMS Health Check nie powiódł się", "error", err)
		os.Exit(1)
	}

	// 1. Pobranie klucza publicznego do weryfikacji tokenów JWT użytkowników
	log.Info("🔑 Pobieranie klucza publicznego JWT z KMS...")
	pubKey, err := kms.FetchPublicKey(ctx, kmsCfg, "shared-jwt")
	if err != nil {
		log.Error("❌ KMS Public Key fetch failed", "error", err)
		os.Exit(1)
	}
	appCfg.JWT.AccessPublicKey = pubKey
	log.Info("✅ KMS Public Key loaded successfully")

	// 2. Pobranie dedykowanych kluczy HMAC dla poszczególnych mikrousług
	for serviceID, target := range appCfg.HMAC.TargetKeys {
		hmacKey, version, err := kms.FetchSymmetricKeyWithVersion(ctx, kmsCfg, target.TargetKey, 1, target.Algorithm)
		if err != nil {
			log.Error("❌ Nie udało się pobrać klucza HMAC z KMS", "service", serviceID, "target_key", target.TargetKey, "error", err)
			os.Exit(1)
		}

		keyStore.SetKey(serviceID, hmacKey, uint32(version))
		log.Info("✅ Klucz HMAC załadowany", "service", serviceID, "version", version)
	}

	// 3. Pobranie klucza HMAC dla RabbitMQ Publishera w Gateway (tylko jeśli RabbitMQ jest włączony)
	var publisherHMACKey []byte
	if appCfg.RabbitMQ.Enabled {
		log.Info("RabbitMQ jest ENABLED. Pobieranie klucza HMAC dla Gateway Publisher...")
		rabbitTarget := appCfg.HMAC.RabbitMQKey
		publisherHMACKey, _, err = kms.FetchSymmetricKeyWithVersion(
			ctx,
			kmsCfg,
			rabbitTarget.TargetKey,
			1,
			rabbitTarget.Algorithm,
		)
		if err != nil {
			log.Error("❌ Nie udało się pobrać klucza HMAC RabbitMQ dla Gateway z KMS", "error", err)
			os.Exit(1)
		}
	}

	return publisherHMACKey
}

func main() {
	// 0. Bootstrap logger setup
	bootLog := shared.InitBootstrapLogger(os.Getenv("ENV"), false)
	defer func() { _ = bootLog.Sync() }()

	// 1. Load configuration
	if err := config.LoadConfigGlobal(); err != nil {
		bootLog.Error("Config load failed", "error", err)
		os.Exit(1)
	}

	log := shared.InitLogger(config.AppConfig.Server.Env, false)

	// Instancjonujemy magazyn kluczy HMAC w pamięci RAM
	keyStore := hmac.NewGatewayKeyStore()

	// Konfiguracja połączenia z KMS
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// =========================================================================
	// 2. KMS SETUP & FETCH KEYS (Jedno czyste wywołanie)
	// =========================================================================
	publisherHMACKey := loadSecurityKeys(ctx, &config.AppConfig, keyStore)

	// 4. Telemetry setup
	if config.AppConfig.OTEL.Enabled {
		cleanup := telemetry.InitTracer(
			config.AppConfig.Server.AppName,
			config.AppConfig.OTEL.Endpoint,
		)
		defer cleanup()
	}

	// 5. Redis connection
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

	// =========================================================================
	// 6. RabbitMQ Event Publisher
	// =========================================================================
	var eventPublisher rabbitmq.EventPublisher
	if config.AppConfig.RabbitMQ.Enabled {
		eventPublisher, err = rabbitmq.NewLivePublisher(
			config.AppConfig.RabbitMQ.GetURL(),
			config.AppConfig.Server.AppName,
			publisherHMACKey,
		)
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

	// 7. DI Container & App initialization
	container := di.NewContainer(redisClient, eventPublisher, &config.AppConfig, keyStore)

	gatewayApp, err := app.NewGatewayApp(container)
	if err != nil {
		log.Error("App setup failed", "error", err)
		os.Exit(1)
	}

	router.SetupRoutes(gatewayApp, container)

	// 8. Start server
	server.Run(
		gatewayApp,
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
