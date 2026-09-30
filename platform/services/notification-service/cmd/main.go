package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/utils"
	"github.com/zerodayz7/platform/services/notification-service/app"
	"github.com/zerodayz7/platform/services/notification-service/config"
	"github.com/zerodayz7/platform/services/notification-service/internal/di"
	"github.com/zerodayz7/platform/services/notification-service/internal/router"
	"github.com/zerodayz7/platform/services/notification-service/internal/security"
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

	// Initialize Redis (optional for placeholder notifications)
	var (
		redisClient *redis.Client
		err        error
	)
	if config.AppConfig.RedisEnabled {
		redisClient, err = redis.New(redis.Config(config.AppConfig.Redis))
		if err != nil {
			log.ErrorObj("Redis failed", err)
		} else {
			defer redisClient.Close()
		}
	} else {
		log.Warn("Redis is disabled for notification-service; worker and stream consumer are off for now.")
	}

	keyStore := httpserver.NewKeyStore()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	securityCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if _, err := security.LoadSecurityKeys(securityCtx, &config.AppConfig, keyStore); err != nil {
		log.Error("❌ Nie udało się załadować kluczy bezpieczeństwa z KMS", "error", err)
		os.Exit(1)
	}

	// Initialize Database
	db, closeDB := config.MustInitDB(config.AppConfig.Database)
	defer closeDB()

	// Dependency Injection setup
	container := di.NewContainer(db, redisClient, log, &config.AppConfig, keyStore)

	container.Workers.NotificationWorker.SetEnabled(config.AppConfig.NotificationWorkerEnabled)

	if config.AppConfig.NotificationWorkerEnabled && config.AppConfig.RedisEnabled {
		utils.SafeGo(log, container.Workers.NotificationWorker.Start)
	} else {
		log.Warn("NotificationWorker is disabled (NOTIFICATION_WORKER_ENABLED=false or REDIS_ENABLED=false). Using DB-seeded placeholder notifications for now.")
	}

	// Initialize Fiber app and routes
	app := app.NewNotificationApp(container)
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
			if redisClient != nil {
				_ = redisClient.Close()
			}
			// Additional resource cleanup can be added here
		},
	)
}
