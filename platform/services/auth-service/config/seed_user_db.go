package config

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/lib/pq"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/auth-service/internal/model"
)

func SeedData(db *gorm.DB) error {
	log := shared.GetLogger()

	var count int64
	db.Model(&model.User{}).Count(&count)

	if count == 0 {
		log.Info("Baza danych jest pusta, rozpoczynam zasiewanie (seeding)...")

		// Hasło do testów dla wszystkich kont: Zaq1@wsx
		testPassword := "WmixUVuBuUWGZzbV2lmXrA$0okaGZfyu+EJgNRSI6aSIyB+WvMDFiBKyN0P+DW7294"

		users := []model.User{
			{
				Username:         "root@plus.pl",
				Email:            "root@plus.pl",
				Password:         testPassword,
				Role:             model.RoleRoot,
				Permissions:      pq.StringArray{model.PermSystemAdmin, model.PermSystemManage, model.PermUsersRead, model.PermUsersWrite, model.PermUsersDelete},
				TwoFactorEnabled: true,
			},
			{
				Username:         "admin@plus.pl",
				Email:            "admin@plus.pl",
				Password:         testPassword,
				Role:             model.RoleAdmin,
				Permissions:      pq.StringArray{model.PermUsersRead, model.PermUsersWrite, model.PermReportsView, model.PermReportsExport},
				TwoFactorEnabled: true,
			},
			{
				Username:         "user@example.com",
				Email:            "user@example.com",
				Password:         testPassword,
				Role:             model.RoleUser,
				Permissions:      pq.StringArray{model.PermReportsView},
				TwoFactorEnabled: true,
			},
		}

		for _, u := range users {
			if err := db.Create(&u).Error; err != nil {
				return fmt.Errorf("failed to seed user %s: %w", u.Username, err)
			}
			log.Info(fmt.Sprintf("[SEED] Utworzono użytkownika: %-18s | Rola: %-6s | Uprawnienia: %v", u.Username, u.Role, u.Permissions))
		}

		log.Info("Seeding zakończony sukcesem.")
	}

	return nil
}
