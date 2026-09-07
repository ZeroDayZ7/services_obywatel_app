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

	// 3. Telemetry (Tracer)
	if config.AppConfig.OTEL.Enabled {
		cleanup := telemetry.InitTracer(
			config.AppConfig.Server.AppName,
			config.AppConfig.OTEL.Endpoint,
		)
		defer cleanup()
	}

	// =========================================================================
	// 4. ZBIORCZY BOOTSTRAP POŚWIADCZEŃ Z SIDECARA NA PODSTAWIE MANIFESTU
	// =========================================================================
	agentManifest, err := agent.LoadManifest("secrets.yaml")
	if err != nil {
		log.Error("❌ Nie udało się wczytać manifestu agenta (secrets.yaml)", "error", err)
		os.Exit(1)
	}

	requiredServices := agentManifest.GetEnabledResourceNames()
	if len(requiredServices) == 0 {
		log.Warn("Brak aktywnych zasobów w manifeście do pobrania podczas bootstrapu")
	}

	log.Info("🚀 Rozpoczynanie zbiorczego bootstrapu poświadczeń",
		"service", agentManifest.Service,
		"resources", requiredServices,
		"socket_path", agentManifest.SocketPath,
	)

	agentCfg := agent.Config{
		SocketPath:    agentManifest.SocketPath,
		TargetService: agentManifest.Service,
		Timeout:       agentManifest.Timeout,
	}

	ctxBootstrap, cancelBootstrap := context.WithTimeout(context.Background(), agentManifest.Timeout)
	bootResp, cleanupSecrets, err := agent.BootstrapApp(ctxBootstrap, agentCfg, requiredServices)
	cancelBootstrap()

	if err != nil {
		log.Error("❌ Zbiorczy bootstrap poświadczeń z sidecara nie powiódł się", "error", err)
		os.Exit(1)
	}

	// Przypisanie poświadczeń do konfiguracji aplikacji przed wyczyszczeniem z pamięci
	if bootResp.Postgres != nil {
		log.Info("✅ Pomyślnie pobrano poświadczenia Postgres", "user", bootResp.Postgres.Username)
		config.AppConfig.Database.User = bootResp.Postgres.Username
		config.AppConfig.Database.Password = string(bootResp.Postgres.Password)
	}

	if bootResp.Redis != nil {
		log.Info("✅ Pomyślnie pobrano poświadczenia Redis", bootResp.Redis.Username)
		if bootResp.Redis.Username != "" {
			config.AppConfig.Redis.Username = bootResp.Redis.Username
		}
		config.AppConfig.Redis.Password = string(bootResp.Redis.Password)
	}

	if bootResp.RabbitMQ != nil {
		log.Info("✅ Pomyślnie pobrano poświadczenia RabbitMQ", "user", bootResp.RabbitMQ.Username)
		config.AppConfig.RabbitMQ.User = bootResp.RabbitMQ.Username
		config.AppConfig.RabbitMQ.Password = string(bootResp.RabbitMQ.Password)
	}

	// -------------------------------------------------------------------------
	// INICJALIZACJA USŁUG Z POBRANYMI POŚWIADCZENIAMI
	// -------------------------------------------------------------------------

	// A. Redis Client Init
	var redisClient *redis.Client
	if bootResp.Redis != nil {
		redisClient, err = redis.New(redis.Config(config.AppConfig.Redis))
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
	db, closeDB := config.MustInitDB(config.AppConfig.Database)
	defer closeDB()

	// CZYŚCIMY PAMIĘĆ Z SUROWYCH BAJTÓW HASEŁ NATYCHMIAST PO POŁĄCZENIU Z USŁUGAMI
	cleanupSecrets()

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

	// Uruchomienie pętli rotacji w goroutines
	if err := agent.StartAutoRotation(ctxApp, agentManifest, gormAdapter, redisAdapter); err != nil {
		log.Error("❌ Nie udało się uruchomić automatycznej rotacji poświadczeń", "error", err)
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
