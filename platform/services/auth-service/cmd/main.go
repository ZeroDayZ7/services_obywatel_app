package main

import (
	"fmt"

	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/telemetry"
	"github.com/zerodayz7/platform/services/auth-service/config"
	"github.com/zerodayz7/platform/services/auth-service/internal/di"
	"github.com/zerodayz7/platform/services/auth-service/internal/router"
)

func main() {
	// 1. Config
	if err := config.LoadConfigGlobal(); err != nil {
		panic(fmt.Sprintf("Config load failed: %v", err))
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

	// 4. Infrastruktura (Redis & DB)
	redisClient, err := redis.New(redis.Config(config.AppConfig.Redis))
	if err != nil {
		log.ErrorObj("Redis failed", err)
	}
	defer redisClient.Close()

	db, closeDB := config.MustInitDB(config.AppConfig.Database)
	defer closeDB()

	// 5. DI & App
	container := di.NewContainer(db, redisClient, &config.AppConfig)
	app := config.NewAuthApp(container)

	// 6. Routes
	router.SetupRoutes(app, container)
	// 7. Graceful shutdown
	server.SetupGracefulShutdown(
		app,
		config.AppConfig.Shutdown,
		func() { closeDB() },
		func() { _ = redisClient.Close() },
	)

	address := "0.0.0.0:" + config.AppConfig.Server.Port
	log.Info("Server started", map[string]any{
		"app":     config.AppConfig.Server.AppName,
		"address": address,
	})
	// Start serwera
	if err := app.Listen(address); err != nil {
		log.ErrorObj("Failed to start server", err)
	}
}
