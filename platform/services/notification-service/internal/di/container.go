package di

import (
	notificationHandler "github.com/zerodayz7/platform/services/notification-service/internal/handler"
	notificationRepo "github.com/zerodayz7/platform/services/notification-service/internal/repository/database"
	notificationService "github.com/zerodayz7/platform/services/notification-service/internal/service"

	"gorm.io/gorm"
)

// Container przechowuje wszystkie zależności serwisów i handlerów
type Container struct {
	NotificationHandler *notificationHandler.NotificationHandler
}

// NewContainer tworzy nowy kontener z wszystkimi zależnościami
func NewContainer(db *gorm.DB) *Container {
	// repozytorium powiadomień
	notificationRepository := notificationRepo.NewNotificationRepository(db)

	// serwis powiadomień
	notificationSvc := notificationService.NewNotificationService(notificationRepository)

	// handler powiadomień
	return &Container{
		NotificationHandler: notificationHandler.NewNotificationHandler(notificationSvc),
	}
}
