package main

import (
	"os"

	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/utils"
	"github.com/zerodayz7/platform/services/notification-service/config"
	"github.com/zerodayz7/platform/services/notification-service/internal/di"
	"github.com/zerodayz7/platform/services/notification-service/internal/router"
	// "github.com/zerodayz7/platform/pkg/telemetry" // Uncomment when telemetry is used
)

func main() {
	// Bootstrap logger for startup errors
	bootLog := shared.InitBootstrapLogger(os.Getenv("ENV"), false)
	defer func() { _ = bootLog.Sync() }()

	// Load global configuration
	if err := config.LoadConfigGlobal(); err != nil {
		bootLog.Fatal("Config load failed", "error", err)
	}

	// Initialize production logger
	log := shared.InitLogger(config.AppConfig.Server.Env, false)

	// Telemetry (Tracer) - Keep commented out as requested
	// if config.AppConfig.OTEL.Enabled {
	// 	cleanup := telemetry.InitTracer(
	// 		config.AppConfig.Server.AppName,
	// 		config.AppConfig.OTEL.Endpoint,
	// 	)
	// 	defer cleanup()
	// }

	// Initialize Redis
	redisClient, err := redis.New(redis.Config(config.AppConfig.Redis))
	if err != nil {
		log.ErrorObj("Redis failed", err)
	}
	defer redisClient.Close()

	// Initialize Database
	db, closeDB := config.MustInitDB(config.AppConfig.Database)
	defer closeDB()

	// Dependency Injection setup
	container := di.NewContainer(db, redisClient, log, &config.AppConfig)

	// Start background workers
	utils.SafeGo(log, container.Workers.NotificationWorker.Start)

	// Initialize Fiber app and routes
	app := config.NewNotificationApp(container)
	router.SetupRoutes(app, container)

	// Start server with unified run handler
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
			closeDB()
			_ = redisClient.Close()
			// Additional resource cleanup can be added here
		},
	)
}
