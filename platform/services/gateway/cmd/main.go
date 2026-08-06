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
	"github.com/zerodayz7/platform/services/gateway/internal/router"
)

func main() {
	// 0. Bootstrap logger setup
	bootLog := shared.InitBootstrapLogger(os.Getenv("ENV"), false)
	defer func() { _ = bootLog.Sync() }()

	// 1. Load configuration
	if err := config.LoadConfigGlobal(); err != nil {
		bootLog.Error("Config load failed", "error", err)
		os.Exit(1)
	}

	// 2. KMS setup & Public Key fetch
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	kmsCfg := kms.Config{
		Endpoint:      config.AppConfig.KMS.Endpoint,
		ServiceName:   config.AppConfig.Server.AppName,
		ServiceSecret: config.AppConfig.KMS.InternalSecret,
	}

	bootLog.Info("🔍 Checking KMS service health...")
	if err := kms.HealthCheck(ctx, kmsCfg); err != nil {
		bootLog.Error("❌ KMS Health Check failed", "error", err)
		os.Exit(1)
	}

	pubKey, err := kms.FetchPublicKey(ctx, kmsCfg, "shared-jwt")
	if err != nil {
		bootLog.Error("❌ KMS Public Key fetch failed", "error", err)
		os.Exit(1)
	}

	config.AppConfig.JWT.AccessPublicKey = pubKey
	bootLog.Info("✅ KMS Public Key loaded successfully")

	// 3. Application logger
	log := shared.InitLogger(config.AppConfig.Server.Env, false)

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

	// 6. RabbitMQ event publisher
	var eventPublisher rabbitmq.EventPublisher
	if config.AppConfig.RabbitMQ.Enabled {
		log.Info("RabbitMQ is ENABLED. Connecting...")
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

	// 7. DI Container & App initialization
	container := di.NewContainer(redisClient, eventPublisher, &config.AppConfig)

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
