package main

import (
	"os"

	"github.com/zerodayz7/platform/pkg/rabbitmq"
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/telemetry"
	"github.com/zerodayz7/platform/services/gateway/config"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
	"github.com/zerodayz7/platform/services/gateway/internal/router"
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

	// 5. RabbitMQ
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

	// 6. DI & App Setup
	container := di.NewContainer(redisClient, eventPublisher, &config.AppConfig)

	app, err := config.NewGatewayApp(container)
	if err != nil {
		log.Error("App setup failed", "error", err)
		os.Exit(1)
	}

	router.SetupRoutes(app, container)

	// 7. Run server
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

			if err := redisClient.Close(); err != nil {
				log.Error("Failed to close Redis client", "error", err)
			}

			if err := eventPublisher.Close(); err != nil {
				log.Error("Failed to close RabbitMQ connection cleanly", "error", err)
			}
		},
	)
}
