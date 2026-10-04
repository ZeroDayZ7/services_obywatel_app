package main

import (
	"context"
	"os"
	"time"

	"github.com/zerodayz7/platform/pkg/agent"
	"github.com/zerodayz7/platform/pkg/database"
	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/rabbitmq"
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/secretprovider"
	"github.com/zerodayz7/platform/pkg/server"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/telemetry"
	"github.com/zerodayz7/platform/pkg/utils"
	"github.com/zerodayz7/platform/services/auth-service/app"
	"github.com/zerodayz7/platform/services/auth-service/config"
	"github.com/zerodayz7/platform/services/auth-service/internal/di"
	"github.com/zerodayz7/platform/services/auth-service/internal/router"
	"github.com/zerodayz7/platform/services/auth-service/internal/security"
)

// #region main
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

	// 3. Telemetry (Tracer)
	if config.AppConfig.OTEL.Enabled {
		cleanup := telemetry.InitTracer(
			config.AppConfig.Server.AppName,
			config.AppConfig.OTEL.Endpoint,
		)
		defer cleanup()
	}

	// =========================================================================
	// 4. SELECT SECRET PROVIDER BEFORE RESOLVING RUNTIME CREDENTIALS
	// =========================================================================
	secretProvider, err := secretprovider.New(secretprovider.Config{
		Provider:      config.AppConfig.Secret.Provider,
		ManifestPath:  "secrets.yaml",
		TargetService: config.AppConfig.Server.AppName,
		Timeout:       5 * time.Second,
	})
	if err != nil {
		log.Error("❌ secret provider initialization failed", "error", err)
		os.Exit(1)
	}
	log.Info("✅ secret provider initialized", "provider", config.AppConfig.Secret.Provider)

	// Runtime bootstrap credentials are materialized locally and kept separate from
	// static config. This avoids mutating AppConfig with secrets while still allowing
	// client initialization to consume the resolved credentials.
	runtimeDB, runtimeRedis, _, err := config.BuildRuntimeConfigs(config.AppConfig, secretProvider)
	if err != nil {
		log.Error("❌ failed to resolve runtime credentials", "error", err)
		os.Exit(1)
	}

	if config.AppConfig.Secret.Provider == secretprovider.KMS12Provider {
		log.Info("✅ running with KMS12 Secret Agent provider")
	}

	// -------------------------------------------------------------------------
	// INICJALIZACJA USŁUG Z POBRANYMI POŚWIADCZENIAMI
	// -------------------------------------------------------------------------
	var agentManifest *agent.Manifest
	if config.AppConfig.Secret.Provider == secretprovider.KMS12Provider {
		agentManifest, err = agent.LoadManifest("secrets.yaml")
		if err != nil {
			log.Error("❌ Nie udało się wczytać manifestu agenta (secrets.yaml)", "error", err)
			os.Exit(1)
		}
	}

	// A. Redis Client Init
	var redisClient *redis.Client
	if runtimeRedis.Password != "" || runtimeRedis.Username != "" {
		redisClient, err = redis.New(redis.Config(runtimeRedis))
		if err != nil || redisClient == nil {
			log.Error("❌ Inicjalizacja Redisa nie powiodła się po pobraniu poświadczeń", "error", err)
			os.Exit(1)
		}
		defer func() {
			if err := redisClient.Close(); err != nil {
				log.Error("Failed to close Redis client", "error", err)
			}
		}()
		log.Info("✅ Połączenie z Redisem nawiązane")
	} else {
		log.Warn("⚠️ Brak poświadczeń Redisa w paczce bootstrap – pomijam inicjalizację Redisa")
	}

	// B. Database Init
	db, closeDB := config.MustInitDB(runtimeDB)
	defer closeDB()

	if cleaner, ok := secretProvider.(interface{ Cleanup() }); ok {
		cleaner.Cleanup()
	}

	// =========================================================================
	// 5. START LICZNIKA / PĘTLI ROTACJI W TLE (Zero-Downtime Credential Rotation)
	// =========================================================================
	// Kontekst dla goroutines rotacji w tle – anulowany dopiero przy zamknięciu aplikacji
	ctxApp, cancelApp := context.WithCancel(context.Background())
	defer cancelApp()

	// Adapter bazy danych Postgres (GORM)
	gormAdapter := database.NewGormAdapter(db)

	// Opcjonalny adapter dla Redisa (jeśli Redis jest włączony)
	var redisAdapter agent.RedisRotatable
	if redisClient != nil {
		redisAdapter = redis.NewAdapter(redisClient)
	}

	if config.AppConfig.Secret.Provider == secretprovider.KMS12Provider && agentManifest != nil {
		if err := agent.StartAutoRotation(ctxApp, agentManifest, gormAdapter, redisAdapter); err != nil {
			log.Error("❌ Nie udało się uruchomić automatycznej rotacji poświadczeń", "error", err)
		}
	}
	// =========================================================================
	// 6. RabbitMQ Publisher Setup
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

	// =========================================================================
	// 7. DI Container & App Setup
	// =========================================================================
	container := di.NewContainer(db, redisClient, eventPublisher, &config.AppConfig, keyStore)
	authApp := app.NewAuthApp(container)

	// =========================================================================
	// 8. RABBITMQ CONSUMERS / WORKERS
	// =========================================================================
	consumerCtx, cancelConsumers := context.WithCancel(context.Background())
	defer cancelConsumers()

	if config.AppConfig.RabbitMQ.Enabled {
		log.Info("🐰 Uruchamianie konsumera RabbitMQ dla rejestracji obywateli...",
			"queue", rabbitmq.QueueAuthCitizen,
			"topic", rabbitmq.TopicCitizenCreated,
		)

		// Run subscriber in SafeGo to protect against goroutine panics
		utils.SafeGo(log, func() {
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
		})
	} else {
		log.Warn("RabbitMQ jest wyłączony - konsumery w tle nie zostały uruchomione.")
	}

	router.SetupRoutes(authApp, container)

	// =========================================================================
	// 9. Run HTTP Server
	// =========================================================================
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
