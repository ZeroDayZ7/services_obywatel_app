package di

import (
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/notification-service/internal/notification"
)

type Workers struct {
	NotificationWorker *notification.NotificationWorker
}

func NewWorkers(redisClient *redis.Client, services *Services, log *shared.Logger) *Workers {
	return &Workers{
		NotificationWorker: notification.NewNotificationWorker(redisClient, services.NotificationSvc, log),
	}
}
