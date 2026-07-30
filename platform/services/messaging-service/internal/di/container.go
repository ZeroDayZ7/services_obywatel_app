package di

import (
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/messaging-service/config"
	"github.com/zerodayz7/platform/services/messaging-service/internal/handler"
	"github.com/zerodayz7/platform/services/messaging-service/internal/repository"
	"github.com/zerodayz7/platform/services/messaging-service/internal/service"
	internalWs "github.com/zerodayz7/platform/services/messaging-service/internal/websocket"
	"gorm.io/gorm"
)

type Container struct {
	DB               *gorm.DB
	Config           *config.Config
	Logger           *shared.Logger
	WsHub            *internalWs.Hub
	MessagingSvc     service.MessagingService
	MessagingHandler *handler.MessagingHandler
}

func NewContainer(db *gorm.DB, logger *shared.Logger, cfg *config.Config, wsHub *internalWs.Hub) *Container {
	messagingRepo := repository.NewMessagingRepository(db)
	messagingSvc := service.NewMessagingService(messagingRepo, cfg, logger)
	messagingHandler := handler.NewMessagingHandler(messagingSvc)

	return &Container{
		DB:               db,
		Config:           cfg,
		Logger:           logger,
		WsHub:            wsHub,
		MessagingSvc:     messagingSvc,
		MessagingHandler: messagingHandler,
	}
}