package config

import (
	"github.com/zerodayz7/platform/services/messaging-service/internal/model"
	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		// E2EE & Device Management
		&model.UserDeviceIdentity{},
		&model.UserPreKey{},

		// Contacts Domain
		&model.Contact{},

		// Chat & Messaging Domain
		&model.Conversation{},
		&model.ConversationMember{},
		&model.Message{},
	)
}
