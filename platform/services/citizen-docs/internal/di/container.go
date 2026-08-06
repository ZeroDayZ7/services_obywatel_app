package di

import (
	"github.com/zerodayz7/platform/pkg/envelope"
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
	UserDocumentSvc service.UserDocumentService
	CitizenSvc      service.CitizenService
}

func NewContainer(
	db *gorm.DB,
	logger *shared.Logger,
	cfg *config.Config,
	cryptor *envelope.EnvelopeCryptor,
) *Container {
	docRepo := repository.NewUserDocumentRepository(db)
	citizenRepo := repository.NewCitizenRepository(db)

	docSvc := service.NewUserDocumentService(docRepo, citizenRepo, cfg, logger, cryptor)
	citizenSvc := service.NewCitizenService(citizenRepo, cfg, logger, cryptor)

	return &Container{
		DB:              db,
		Config:          cfg,
		Logger:          logger,
		Cryptor:         cryptor,
		UserDocumentSvc: docSvc,
		CitizenSvc:      citizenSvc,
	}
}
