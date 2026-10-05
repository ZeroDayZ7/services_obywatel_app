package di

import (
	"github.com/zerodayz7/platform/pkg/envelope"
	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/citizen-docs/config"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/repository"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/service"
	"gorm.io/gorm"
)

type Container struct {
	DB              *gorm.DB
	Config          *config.Config
	Logger          *shared.Logger
	Cryptor         *envelope.EnvelopeCryptor
	KeyStore        *httpserver.KeyStore
	UserDocumentSvc service.UserDocumentService
}

func NewContainer(
	db *gorm.DB,
	logger *shared.Logger,
	cfg *config.Config,
	cryptor *envelope.EnvelopeCryptor,
	keyStore *httpserver.KeyStore,
) *Container {
	hmacDocumentNumberSecret, _, ok := keyStore.GetKey("document_number")
	if !ok {
		panic("critical error: missing 'document_number' hmac key in KeyStore")
	}

	docRepo := repository.NewUserDocumentRepository(db)
	docSvc := service.NewUserDocumentService(docRepo, cfg, cryptor, hmacDocumentNumberSecret, "documents-metadata-key")

	return &Container{
		DB:              db,
		Config:          cfg,
		Logger:          logger,
		Cryptor:         cryptor,
		KeyStore:        keyStore,
		UserDocumentSvc: docSvc,
	}
}
