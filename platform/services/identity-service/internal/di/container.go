package di

import (
	"fmt"

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

//#region BuildContainer
func BuildContainer(
	app *config.App,
	eventPublisher rabbitmq.EventPublisher,
	kmsCfg kms.Config,
	keyStore *httpserver.KeyStore,
	fileStorage storage.StorageClient,
) *Container {
	// 1. Klucze HMAC do bezpiecznego hashowania (indeksowania) danych
	hmacPeselSecret, _, ok := keyStore.GetKey("pesel")
	if !ok {
		panic("critical error: missing 'pesel' hmac key in KeyStore")
	}

	hmacPhoneSecret, _, ok := keyStore.GetKey("phone")
	if !ok {
		panic("critical error: missing 'phone' hmac key in KeyStore")
	}

	hmacEmailSecret, _, ok := keyStore.GetKey("email")
	if !ok {
		panic("critical error: missing 'email' hmac key in KeyStore")
	}

	auditHmacKey, _, ok := keyStore.GetKey("audit")
	if !ok {
		panic("critical error: missing 'audit' key in KeyStore")
	}

	hmacPukSecret, _, ok := keyStore.GetKey("puk")
	if !ok {
		panic("critical error: missing 'puk' hmac key in KeyStore")
	}

	// Repozytoria
	citizenRepo := repository.NewCitizenRepository(app.DB, auditHmacKey)
	outboxRepo := repository.NewOutboxRepository(app.DB)

	cryptor := envelope.NewEnvelopeCryptor(kmsCfg)

	pdfGen, err := service.NewPDFGenerator()
	if err != nil {
		panic(fmt.Sprintf("critical error: failed to initialize PDF generator: %v", err))
	}

	// 2. Przekazanie osobnych kluczy HMAC do serwisu obywatela
	citizenSvc := service.NewCitizenService(
		citizenRepo,
		cryptor,
		fileStorage,
		pdfGen,
		hmacPeselSecret,
		hmacPhoneSecret,
		hmacEmailSecret,
		hmacPukSecret,
		"identity-citizen-key",
		"identity-agreements-key",
		"identity-phone-key",
		"identity-email-key",
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
