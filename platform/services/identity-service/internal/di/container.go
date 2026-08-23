package di

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zerodayz7/platform/pkg/envelope"
	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/kms"
	"github.com/zerodayz7/platform/pkg/rabbitmq"
	"github.com/zerodayz7/platform/pkg/storage"
	"github.com/zerodayz7/services/identity-service/config"
	"github.com/zerodayz7/services/identity-service/internal/handler"
	"github.com/zerodayz7/services/identity-service/internal/repository"
	"github.com/zerodayz7/services/identity-service/internal/service"
)

type Container struct {
	Config         *config.Config
	DB             *pgxpool.Pool
	EventPublisher rabbitmq.EventPublisher
	CitizenHandler *handler.CitizenHandler
	KeyStore       *httpserver.KeyStore
	OutboxRepo     repository.OutboxRepository
	Storage        storage.StorageClient
}

func BuildContainer(
	app *config.App,
	eventPublisher rabbitmq.EventPublisher,
	kmsCfg kms.Config,
	keyStore *httpserver.KeyStore,
	fileStorage storage.StorageClient,
) *Container {
	peselHmacKey, _, ok := keyStore.GetKey("pesel")
	if !ok {
		panic("critical error: missing 'pesel' key in KeyStore")
	}

	auditHmacKey, _, ok := keyStore.GetKey("audit")
	if !ok {
		panic("critical error: missing 'audit' key in KeyStore")
	}

	// Repozytoria
	citizenRepo := repository.NewCitizenRepository(app.DB, auditHmacKey)
	outboxRepo := repository.NewOutboxRepository(app.DB)

	cryptor := envelope.NewEnvelopeCryptor(kmsCfg)

	// Serwisy
	citizenSvc := service.NewCitizenService(
		citizenRepo,
		cryptor,
		fileStorage,
		peselHmacKey,
		"identity-citizen-data",
		"identity-agreements-key",
	)

	// Handlery
	citizenHdl := handler.NewCitizenHandler(citizenSvc)

	return &Container{
		Config:         app.Config,
		DB:             app.DB,
		EventPublisher: eventPublisher,
		CitizenHandler: citizenHdl,
		KeyStore:       keyStore,
		OutboxRepo:     outboxRepo,
		Storage:        fileStorage,
	}
}
