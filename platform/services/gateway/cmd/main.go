package main

import (
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
	// 0. Boostrap Logger
	bootLog := shared.InitBootstrapLogger(os.Getenv("ENV"))
	defer func() { _ = bootLog.Sync() }()

	// 1. Config
	if err := config.LoadConfigGlobal(); err != nil {
		bootLog.Fatal("Config load failed", "error", err)
	}

	// 2. Logger
	log := shared.InitLogger(config.AppConfig.Server.Env)

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
		log.ErrorObj("Redis failed", err)
	}
	defer redisClient.Close()

	// 5. DI & App Setup
	container := di.NewContainer(redisClient, &config.AppConfig)
	app := config.NewGatewayApp(container)
	router.SetupRoutes(app, container)

	// 6. Run server with unified shutdown handler
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
			// Additional resource cleanup (e.g., database) can be added here in the future.
		},
	)
}
