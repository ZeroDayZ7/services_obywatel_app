package di

import (
	"github.com/zerodayz7/platform/pkg/httpserver"
	pkgredis "github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/storage"
	"github.com/zerodayz7/platform/services/messaging-service/config"
	"github.com/zerodayz7/platform/services/messaging-service/internal/handler"
	pub "github.com/zerodayz7/platform/services/messaging-service/internal/publisher"
	"github.com/zerodayz7/platform/services/messaging-service/internal/repository"
	"github.com/zerodayz7/platform/services/messaging-service/internal/service"
	"gorm.io/gorm"
)

type Container struct {
	DB               *gorm.DB
	Config           *config.Config
	Logger           *shared.Logger
	KeyStore         *httpserver.KeyStore
	Storage          storage.StorageClient
	Redis            *pkgredis.Client
	MessagingSvc     service.MessagingService
	ContactsSvc      service.ContactsService
	MessagingHandler *handler.MessagingHandler
	ContactsHandler  *handler.ContactsHandler
}

func NewContainer(
	db *gorm.DB,
	redisClient *pkgredis.Client,
	fileStorage storage.StorageClient,
	logger *shared.Logger,
	cfg *config.Config,
	keyStore *httpserver.KeyStore,
) *Container {
	messagingRepo := repository.NewMessagingRepository(db)
	contactsRepo := repository.NewContactsRepository(db)
	eventPublisher := pub.NewRedisPublisher(redisClient, logger)

	messagingSvc := service.NewMessagingService(messagingRepo, cfg, logger, eventPublisher)
	contactsSvc := service.NewContactsService(contactsRepo, messagingRepo, logger, eventPublisher)

	messagingHandler := handler.NewMessagingHandler(messagingSvc)
	contactsHandler := handler.NewContactsHandler(contactsSvc)

	return &Container{
		DB:               db,
		Config:           cfg,
		Logger:           logger,
		KeyStore:         keyStore,
		Redis:            redisClient,
		MessagingSvc:     messagingSvc,
		ContactsSvc:      contactsSvc,
		MessagingHandler: messagingHandler,
		ContactsHandler:  contactsHandler,
	}
}
