package main

import (
	"context"
	"os"
	"time"

	"github.com/zerodayz7/platform/pkg/database"
	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/rabbitmq"
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/telemetry"
	"github.com/zerodayz7/platform/services/auth-service/app"
	"github.com/zerodayz7/platform/services/auth-service/config"
	"github.com/zerodayz7/platform/services/auth-service/internal/di"
	"github.com/zerodayz7/platform/services/auth-service/internal/router"
	"github.com/zerodayz7/platform/services/auth-service/internal/security"
)

//#region main
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

	// Instancja RAM KeyStore do przechowywania kluczy dla akceptowanych nadawców
	keyStore := httpserver.NewKeyStore()

	// =========================================================================
	// 2. KMS SETUP & BOOTSTRAP KEYS (JWT Private Key + Internal HMAC Keys)
	// =========================================================================
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rabbitHMACKey, err := security.LoadSecurityKeys(ctx, &config.AppConfig, keyStore)
	if err != nil {
		log.Error("❌ Nie udało się załadować kluczy bezpieczeństwa z KMS", "error", err)
		os.Exit(1)
	}

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
	// =========================================================================
	// 6a. POBIERANIE POŚWIADCZEŃ Z SIDECARA PRZEZ UDS (BEZPOŚREDNIO)
	// =========================================================================
	kmsCfg := shared.Config{
		SocketPath:    config.AppConfig.Agent.SocketPath,
		TargetService: "auth_service",
		Timeout:       5 * time.Second,
	}

	log.Info("🔌 Łączenie z secret-agent przez UDS...", "path", kmsCfg.SocketPath)

	cacheKey := "postgres"

	ctxCreds, cancelCreds := context.WithTimeout(context.Background(), kmsCfg.Timeout)
	dbCreds, cleanup, err := shared.FetchAgentSecret(ctxCreds, kmsCfg, cacheKey)
	cancelCreds()
	// defer cleanup()

	if err != nil {
		log.Error("❌ Nie udało się pobrać poświadczeń DB z sidecara", "error", err)
		os.Exit(1)
	}
	defer cleanup()

	// Weryfikacja poświadczeń DB
	if dbCreds == nil {
		log.Error("❌ Sidecar zwrócił pustą odpowiedź (brak poświadczeń do Postgresa)")
		os.Exit(1)
	}

	log.Info("✅ Pomyślnie pobrano poświadczenia DB z sidecara", "username", dbCreds.Username)

	// Nadpisujemy konfigurację pobranymi danymi
	config.AppConfig.Database.User = dbCreds.Username
	config.AppConfig.Database.Password = string(dbCreds.Password)

	// 6. Database Init
	db, closeDB := config.MustInitDB(config.AppConfig.Database)
	defer closeDB()

	// Zerujemy wrażliwe dane z konfiguracji po ustanowieniu połączenia
	config.AppConfig.Database.Password = ""

	// =========================================================================
	// 6b. GOROUTINE ODŚWIEŻAJĄCA POŚWIADCZENIA W TLE (ROTACJA)
	// =========================================================================
	ctxApp, cancelApp := context.WithCancel(context.Background())
	defer cancelApp()

	go shared.StartPostgresRotationLoop(ctxApp, kmsCfg, 30*time.Minute, database.NewGormAdapter(db))
	// =========================================================================
	// 7. RabbitMQ Publisher Setup
	// =========================================================================
	var eventPublisher rabbitmq.EventPublisher
	if config.AppConfig.RabbitMQ.Enabled {
		log.Info("RabbitMQ is ENABLED. Fetching HMAC key for Publisher...")

		eventPublisher, err = rabbitmq.NewLivePublisher(
			config.AppConfig.RabbitMQ.GetURL(),
			config.AppConfig.Server.AppName,
			rabbitHMACKey,
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

	// 8. DI Container & App Setup (Przekazujemy keyStore)
	container := di.NewContainer(db, redisClient, eventPublisher, &config.AppConfig, keyStore)
	authApp := app.NewAuthApp(container)

	// =========================================================================
	// 8a. RABBITMQ CONSUMERS / WORKERS
	// =========================================================================
	consumerCtx, cancelConsumers := context.WithCancel(context.Background())
	defer cancelConsumers()

	if config.AppConfig.RabbitMQ.Enabled {
		log.Info("🐰 Uruchamianie konsumera RabbitMQ dla rejestracji obywateli...",
			"queue", rabbitmq.QueueAuthCitizen,
			"topic", rabbitmq.TopicCitizenCreated,
		)

		go func() {
			err := eventPublisher.SubscribeWithAuth(
				consumerCtx,
				rabbitmq.QueueAuthCitizen,
				rabbitmq.TopicCitizenCreated,
				container.KeyStore,
				container.Consumers.CitizenConsumer.HandleCitizenCreated,
			)
			if err != nil && consumerCtx.Err() == nil {
				log.Error("❌ Error in citizen created consumer", "error", err)
			}
		}()
	} else {
		log.Warn("⚠️ RabbitMQ jest wyłączony - konsumery w tle nie zostały uruchomione.")
	}

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
