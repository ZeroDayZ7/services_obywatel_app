package main

import (
	"fmt"
	"os"

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

	fmt.Printf("DEBUG: ENV REDIS_HOST: %s\n", os.Getenv("REDIS_HOST"))
	fmt.Printf("DEBUG: VIPER REDIS_HOST: %s\n", config.AppConfig.Redis.Host)

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
		log.Error("Redis failed", "error", err)
		os.Exit(1)
	}

	if redisClient == nil {
		log.Error("Redis client is nil")
		os.Exit(1)
	}

	defer func() {
		_ = redisClient.Close()
	}()

	// 5. DI & App Setup
	container := di.NewContainer(redisClient, &config.AppConfig)

	app, err := config.NewGatewayApp(container)
	if err != nil {
		log.Error("App setup failed", "error", err)
		os.Exit(1)
	}

	router.SetupRoutes(app, container)

	// 6. Run server
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
			_ = redisClient.Close()
		},
	)
}
