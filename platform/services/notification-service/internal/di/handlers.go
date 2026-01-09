package di

import (
	notificationHandler "github.com/zerodayz7/platform/services/notification-service/internal/handler"
)

type Handlers struct {
	NotificationHandler *notificationHandler.NotificationHandler
}

func NewHandlers(services *Services) *Handlers {
	return &Handlers{
		NotificationHandler: notificationHandler.NewNotificationHandler(services.NotificationSvc),
	}
}
