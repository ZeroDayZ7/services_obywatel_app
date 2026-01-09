package di

import (
	notificationService "github.com/zerodayz7/platform/services/notification-service/internal/service"
)

type Services struct {
	NotificationSvc *notificationService.NotificationService
}

func NewServices(repos *Repositories) *Services {
	return &Services{
		NotificationSvc: notificationService.NewNotificationService(repos.NotificationRepo),
	}
}
