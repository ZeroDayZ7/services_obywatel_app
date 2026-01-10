package di

import (
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/repository"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/service"
	"gorm.io/gorm"
)

type Container struct {
	DB              *gorm.DB
	Config          *viper.Config
	Logger          *shared.Logger
	UserDocumentSvc *service.UserDocumentService
	CitizenSvc      *service.CitizenService
}

func NewContainer(db *gorm.DB, logger *shared.Logger, cfg *viper.Config) *Container {
	// Repozytoria
	docRepo := repository.NewUserDocumentRepository(db)
	citizenRepo := repository.NewCitizenRepository(db)

	docSvc := service.NewUserDocumentService(docRepo, cfg, logger)
	citizenSvc := service.NewCitizenService(citizenRepo, cfg, logger)

	return &Container{
		DB:              db,
		Config:          cfg,
		Logger:          logger,
		UserDocumentSvc: docSvc,
		CitizenSvc:      citizenSvc,
	}
}
