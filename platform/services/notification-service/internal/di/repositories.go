package di

import (
	notificationRepo "github.com/zerodayz7/platform/services/notification-service/internal/repository/database"
	"gorm.io/gorm"
)

type Repositories struct {
	NotificationRepo *notificationRepo.NotificationRepository
}

func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		NotificationRepo: notificationRepo.NewNotificationRepository(db),
	}
}
