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
	ContactsSvc      service.ContactsService
	MessagingHandler *handler.MessagingHandler
	ContactsHandler  *handler.ContactsHandler
}

func NewContainer(db *gorm.DB, logger *shared.Logger, cfg *config.Config, wsHub *internalWs.Hub) *Container {
	messagingRepo := repository.NewMessagingRepository(db)
	contactsRepo := repository.NewContactsRepository(db) // Nowy repozytorium

	messagingSvc := service.NewMessagingService(messagingRepo, cfg, logger)
	contactsSvc := service.NewContactsService(contactsRepo, logger)

	messagingHandler := handler.NewMessagingHandler(messagingSvc)
	contactsHandler := handler.NewContactsHandler(contactsSvc)

	return &Container{
		DB:               db,
		Config:           cfg,
		Logger:           logger,
		WsHub:            wsHub,
		MessagingSvc:     messagingSvc,
		ContactsSvc:      contactsSvc,
		MessagingHandler: messagingHandler,
		ContactsHandler:  contactsHandler,
	}
}
