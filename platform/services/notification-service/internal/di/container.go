package di

import (
	"github.com/zerodayz7/platform/pkg/redis"
	notificationHandler "github.com/zerodayz7/platform/services/notification-service/internal/handler"
	"github.com/zerodayz7/platform/services/notification-service/internal/notification"
	notificationRepo "github.com/zerodayz7/platform/services/notification-service/internal/repository/database"
	notificationService "github.com/zerodayz7/platform/services/notification-service/internal/service"

	"github.com/zerodayz7/platform/pkg/shared"
	"gorm.io/gorm"
)

type Container struct {
	NotificationHandler *notificationHandler.NotificationHandler
	NotificationWorker  *notification.NotificationWorker
	Redis               *redis.Client
	Logger              *shared.Logger
}

func NewContainer(db *gorm.DB, redisClient *redis.Client, log *shared.Logger) *Container {
	// repo
	notificationRepository := notificationRepo.NewNotificationRepository(db)

	// serwis
	notificationSvc := notificationService.NewNotificationService(notificationRepository)

	// handler
	handler := notificationHandler.NewNotificationHandler(notificationSvc)

	// worker
	worker := notification.NewNotificationWorker(redisClient, notificationSvc, log)

	return &Container{
		NotificationHandler: handler,
		NotificationWorker:  worker,
		Redis:               redisClient,
		Logger:              log,
	}
}
