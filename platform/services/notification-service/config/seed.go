package config

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/notification-service/internal/model"
	"gorm.io/gorm"
)

// Stały UUID użytkownika testowego (odpowiadający np. użytkownikowi z auth-service/seed)
var testUserID = uuid.MustParse("707a8869-6867-4601-9337-e23fcb51b0ad")

func SeedData(db *gorm.DB) error {
	var count int64
	if err := db.Model(&model.Notification{}).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check notification count: %w", err)
	}

	if count > 0 {
		log.Println("[SEED] Notification table already seeded, skipping...")
		return nil
	}

	log.Println("[SEED] Seeding notification database...")

	notifications := []model.Notification{
		{
			UserID:   testUserID,
			Title:    "Witamy w systemie Obywatel Plus",
			Content:  "Konto zostało pomyślnie aktywowane. Masz dostęp do wszystkich usług cyfrowych.",
			Priority: "info",
			Category: "system",
			IsRead:   false,
		},
		{
			UserID:   testUserID,
			Title:    "Wniosek został rozpatrzony",
			Content:  "Twój wniosek o wydanie dowodu osobistego zmienił status na: GOTOWY DO ODBIORU.",
			Priority: "success",
			Category: "system",
			IsRead:   false,
		},
		{
			UserID:   testUserID,
			Title:    "Przypomnienie o wygasającym dokumencie",
			Content:  "Paszport wygasa za 30 dni. Złóż wniosek online, aby uniknąć opóźnień.",
			Priority: "warning",
			Category: "administrative",
			IsRead:   true,
		},
	}

	if err := db.Create(&notifications).Error; err != nil {
		return fmt.Errorf("failed to seed notifications: %w", err)
	}

	log.Println("[SEED] Notification database seeded successfully!")
	return nil
}
