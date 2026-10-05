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
	if cfg == nil {
		panic("critical error: config is nil")
	}
	if keyStore == nil {
		panic("critical error: keystore is nil")
	}

	const metadataKeyAlias = "docs-id-cards"
	if _, ok := cfg.HMAC.InternalKeys[metadataKeyAlias]; !ok {
		panic("critical error: missing 'docs-id-cards' metadata key in HMAC internal config")
	}

	hmacDocumentNumberSecret, _, ok := keyStore.GetKey("document_number")
	if !ok || len(hmacDocumentNumberSecret) == 0 {
		panic("critical error: missing or empty 'document_number' hmac key in KeyStore")
	}

	docRepo := repository.NewUserDocumentRepository(db)
	docSvc := service.NewUserDocumentService(docRepo, cfg, cryptor, hmacDocumentNumberSecret, metadataKeyAlias)

	return &Container{
		DB:              db,
		Config:          cfg,
		Logger:          logger,
		Cryptor:         cryptor,
		KeyStore:        keyStore,
		UserDocumentSvc: docSvc,
	}
}
