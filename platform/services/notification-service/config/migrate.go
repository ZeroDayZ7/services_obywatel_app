package config

import (
	"github.com/zerodayz7/platform/services/notification-service/internal/model"
	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.Notification{},
	)
}
