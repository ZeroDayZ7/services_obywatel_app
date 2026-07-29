package di

import (
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/messaging-service/config"
	"github.com/zerodayz7/platform/services/messaging-service/internal/handler"
	"github.com/zerodayz7/platform/services/messaging-service/internal/repository"
	"github.com/zerodayz7/platform/services/messaging-service/internal/service"
	"gorm.io/gorm"
)

type Container struct {
	DB               *gorm.DB
	Config           *config.Config
	Logger           *shared.Logger
	MessagingSvc     service.MessagingService
	MessagingHandler *handler.MessagingHandler
}

func NewContainer(db *gorm.DB, logger *shared.Logger, cfg *config.Config) *Container {
	messagingRepo := repository.NewMessagingRepository(db)
	messagingSvc := service.NewMessagingService(messagingRepo, cfg, logger)
	messagingHandler := handler.NewMessagingHandler(messagingSvc)

	return &Container{
		DB:               db,
		Config:           cfg,
		Logger:           logger,
		MessagingSvc:     messagingSvc,
		MessagingHandler: messagingHandler,
	}
}
