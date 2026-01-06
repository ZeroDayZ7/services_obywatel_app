package config

import (
	"fmt"

	"gorm.io/gorm"

	// Zmień te ścieżki na Twoje faktyczne ścieżki w projekcie!
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/auth-service/internal/features/auth/model"
)

func SeedData(db *gorm.DB) error {
	log := shared.GetLogger()

	var count int64
	db.Model(&model.User{}).Count(&count)

	if count == 0 {
		log.Info("Baza danych jest pusta, rozpoczynam zasiewanie (seeding)...")

		// Wspólny hash hasła z Twojego przykładu
		testPassword := "WmixUVuBuUWGZzbV2lmXrA$0okaGZfyu+EJgNRSI6aSIyB+WvMDFiBKyN0P+DW7294"

		users := []model.User{
			{
				Username:         "admin@plus.pl",
				Email:            "admin@plus.pl",
				Password:         testPassword,
				Role:             "admin",
				TwoFactorEnabled: true,
			},
			{
				Username:         "user@example.com",
				Email:            "user@example.com",
				Password:         testPassword,
				Role:             "user",
				TwoFactorEnabled: true,
			},
		}

		// Wstawiamy obu użytkowników w jednej transakcji
		for _, u := range users {
			if err := db.Create(&u).Error; err != nil {
				return fmt.Errorf("failed to seed user %s: %w", u.Username, err)
			}
			log.Info(fmt.Sprintf("Utworzono użytkownika: %s (Role: %s, ID: %s)", u.Username, u.Role, u.ID))
		}

		log.Info("Seeding zakończony sukcesem.")
	}

	return nil
}
