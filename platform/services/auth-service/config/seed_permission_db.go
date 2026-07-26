package config

import (
	"fmt"

	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/auth-service/internal/model"
	"gorm.io/gorm"
)

func SeedPermissions(db *gorm.DB) error {
	log := shared.GetLogger()

	var count int64
	if err := db.Model(&model.AvailablePermission{}).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to count available permissions: %w", err)
	}

	if count > 0 {
		return nil
	}

	log.Info("[SEED] Rozpoczynam zasiewanie tabeli dostępnych uprawnień...")

	permissions := []model.AvailablePermission{
		// Moduł Systemowy
		{
			Key:         model.PermSystemAdmin,
			Department:  "SYSTEM",
			Description: "Pełne uprawnienia administracyjne do całego systemu",
			IsSpecial:   true,
		},
		{
			Key:         model.PermSystemManage,
			Department:  "SYSTEM",
			Description: "Zarządzanie konfiguracją i parametrami systemowymi",
			IsSpecial:   true,
		},

		// Moduł Użytkowników
		{
			Key:         model.PermUsersRead,
			Department:  "USERS",
			Description: "Odczyt listy i szczegółów użytkowników",
			IsSpecial:   false,
		},
		{
			Key:         model.PermUsersWrite,
			Department:  "USERS",
			Description: "Tworzenie oraz edycja kont użytkowników",
			IsSpecial:   false,
		},
		{
			Key:         model.PermUsersDelete,
			Department:  "USERS",
			Description: "Trwałe usuwanie lub deaktywacja użytkowników",
			IsSpecial:   true,
		},

		// Moduł Raportów
		{
			Key:         model.PermReportsView,
			Department:  "REPORTS",
			Description: "Przeglądanie raportów i statystyk systemowych",
			IsSpecial:   false,
		},
		{
			Key:         model.PermReportsExport,
			Department:  "REPORTS",
			Description: "Eksport danych raportowych do plików zewnętrznych",
			IsSpecial:   false,
		},
	}

	for _, p := range permissions {
		if err := db.Create(&p).Error; err != nil {
			return fmt.Errorf("failed to seed permission %s: %w", p.Key, err)
		}
		log.Info(fmt.Sprintf("[SEED] Utworzono uprawnienie: %-20s | Dział: %-10s | Specjalne: %t", p.Key, p.Department, p.IsSpecial))
	}

	return nil
}
