package di

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zerodayz7/platform/pkg/envelope"
	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/kms"
	"github.com/zerodayz7/platform/pkg/rabbitmq"
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
}

func BuildContainer(
	app *config.App,
	eventPublisher rabbitmq.EventPublisher,
	peselHmacKey []byte,
	kmsCfg kms.Config,
	keyStore *httpserver.KeyStore,
) *Container {
	citizenRepo := repository.NewCitizenRepository(app.DB)
	outboxRepo := repository.NewOutboxRepository(app.DB)

	cryptor := envelope.NewEnvelopeCryptor(kmsCfg)

	citizenSvc := service.NewCitizenService(
		citizenRepo,
		cryptor,
		peselHmacKey,
		"identity-citizen-data",
	)

	citizenHdl := handler.NewCitizenHandler(citizenSvc)

	return &Container{
		Config:         app.Config,
		DB:             app.DB,
		EventPublisher: eventPublisher,
		CitizenHandler: citizenHdl,
		KeyStore:       keyStore,
		OutboxRepo:     outboxRepo,
	}
}
