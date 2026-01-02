package main

import (
	"os"

	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/gateway/config"
	"github.com/zerodayz7/platform/services/gateway/internal/di"
	"github.com/zerodayz7/platform/services/gateway/internal/router"
)

func main() {
	// Inicjalizacja loggera
	log := shared.InitLogger(os.Getenv("ENV"))

	// Config
	if err := config.LoadConfigGlobal(); err != nil {
		log.ErrorObj("Config load failed", err)
		return
	}

	// Redis – z nowego pkg
	redisClient, err := redis.New(redis.Config(config.AppConfig.Redis))
	if err != nil {
		log.ErrorObj("Redis failed", err)
	}
	defer redisClient.Close()

	// Cache wrapper bez TTL
	cache := redis.NewCache(redisClient, "session:", 0)

	container := di.NewContainer(
		redisClient,
		cache,
		config.AppConfig.Proxy.RequestTimeout,
		config.AppConfig.Proxy.MaxIdleConns,
		config.AppConfig.Proxy.MaxIdleConnsPerHost,
	)

	// 5. Fiber app (config importuje di - to jest ok)
	app := config.NewGatewayApp(container)

	// Routes
	router.SetupRoutes(app, container)

	// Graceful shutdown
	server.SetupGracefulShutdown(app, nil, config.AppConfig.Shutdown)

	// Log start
	address := "0.0.0.0:" + config.AppConfig.Server.Port
	log.InfoObj("Gateway-Server listening", map[string]any{"address": address})

	// Start serwera
	if err := app.Listen(address); err != nil {
		log.ErrorObj("Failed to start server", err)
	}
}
