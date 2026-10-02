package config

import (
	"fmt"

	"github.com/zerodayz7/platform/pkg/permissions"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/auth-service/internal/model"
	"gorm.io/gorm"
)

// #region SeedPermissions
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

	permissionsList := []model.AvailablePermission{
		// Moduł Systemowy
		{
			Key:         permissions.SystemAdmin,
			Department:  "SYSTEM",
			Description: "Pełne uprawnienia administracyjne do całego systemu",
			IsSpecial:   true,
		},
		{
			Key:         permissions.SystemManage,
			Department:  "SYSTEM",
			Description: "Zarządzanie konfiguracją i parametrami systemowymi",
			IsSpecial:   true,
		},

		// Moduł Użytkowników
		{
			Key:         permissions.UsersRead,
			Department:  "USERS",
			Description: "Odczyt listy i szczegółów użytkowników",
			IsSpecial:   false,
		},
		{
			Key:         permissions.UsersWrite,
			Department:  "USERS",
			Description: "Tworzenie oraz edycja kont użytkowników",
			IsSpecial:   false,
		},
		{
			Key:         permissions.UsersDelete,
			Department:  "USERS",
			Description: "Trwałe usuwanie lub deaktywacja użytkowników",
			IsSpecial:   true,
		},

		// Moduł Raportów
		{
			Key:         permissions.ReportsView,
			Department:  "REPORTS",
			Description: "Przeglądanie raportów i statystyk systemowych",
			IsSpecial:   false,
		},
		{
			Key:         permissions.ReportsExport,
			Department:  "REPORTS",
			Description: "Eksport danych raportowych do plików zewnętrznych",
			IsSpecial:   false,
		},

		// Moduł Wiadomości
		{
			Key:         permissions.MessagesRead,
			Department:  "MESSAGES",
			Description: "Odczyt wiadomości w systemie",
			IsSpecial:   false,
		},
		{
			Key:         permissions.MessagesWrite,
			Department:  "MESSAGES",
			Description: "Wysyłanie i edycja wiadomości",
			IsSpecial:   false,
		},
		{
			Key:         permissions.MessagingAccess,
			Department:  "MESSAGES",
			Description: "Dostęp do modułu komunikatora",
			IsSpecial:   false,
		},

		// Moduł Dokumentów
		{
			Key:         permissions.DocumentsRead,
			Department:  "DOCUMENTS",
			Description: "Odczyt dokumentów obywatelskich i urzędowych",
			IsSpecial:   false,
		},
		{
			Key:         permissions.DocumentsWrite,
			Department:  "DOCUMENTS",
			Description: "Składanie i edycja wniosków/dokumentów",
			IsSpecial:   false,
		},

		// Moduł Tożsamości, Powiadomień i Sesji
		{
			Key:         "identity.me.read",
			Department:  "IDENTITY",
			Description: "Odczyt własnego profilu tożsamości",
			IsSpecial:   false,
		},
		{
			Key:         "notifications.read",
			Department:  "NOTIFICATIONS",
			Description: "Odbiór i odczyt powiadomień push/systemowych",
			IsSpecial:   false,
		},
		{
			Key:         "sessions.manage",
			Department:  "AUTH",
			Description: "Zarządzanie aktywnymi sesjami i urządzeniami",
			IsSpecial:   false,
		},

		// Moduł Urzędniczy / Spraw
		{
			Key:         "cases.manage",
			Department:  "OFFICER",
			Description: "Zarządzanie sprawami obywateli przez urzędnika",
			IsSpecial:   false,
		},
		{
			Key:         "officer.actions",
			Department:  "OFFICER",
			Description: "Wykonywanie akcji decyzyjnych w portalu urzędnika",
			IsSpecial:   false,
		},
	}

	for _, p := range permissionsList {
		if err := db.Create(&p).Error; err != nil {
			return fmt.Errorf("failed to seed permission %s: %w", p.Key, err)
		}
		log.Info(fmt.Sprintf("[SEED] Utworzono uprawnienie: %-20s | Dział: %-10s | Specjalne: %t", p.Key, p.Department, p.IsSpecial))
	}

	return nil
}
