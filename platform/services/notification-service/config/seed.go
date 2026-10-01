package config

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/notification-service/internal/model"
	"gorm.io/gorm"
)

// ID użytkowników z rolą CITIZEN ze skryptu seed_users.go
var (
	adminUserID = uuid.MustParse("707a8869-6867-4601-9337-e23fcb51b0ad") // Admin / Citizen
	janKowalski = uuid.MustParse("a0123456-1111-2222-3333-444455556666") // Jan Kowalski
	annaNowak   = uuid.MustParse("b0123456-1111-2222-3333-444455556666") // Anna Nowak
)

//#region SeedData
func SeedData(db *gorm.DB) error {
	log := shared.GetLogger()

	var count int64
	if err := db.Model(&model.Notification{}).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check notification count: %w", err)
	}

	if count > 0 {
		return nil
	}

	log.Info("[SEED] Rozpoczynam zasiewanie bazy powiadomień dla obywateli...")

	notifications := []model.Notification{
		// --- Admin User (Citizen) ---
		{
			ID:       shared.NewUUIDv7(),
			UserID:   adminUserID,
			Title:    "Witamy w systemie Obywatel Plus",
			Content:  "Konto zostało pomyślnie aktywowane. Masz dostęp do wszystkich usług cyfrowych.",
			Priority: "info",
			Category: "system",
			IsRead:   true,
		},
		{
			ID:       shared.NewUUIDv7(),
			UserID:   adminUserID,
			Title:    "Wniosek został rozpatrzony",
			Content:  "Twój wniosek o wydanie dowodu osobistego zmienił status na: GOTOWY DO ODBIORU.",
			Priority: "success",
			Category: "administrative",
			IsRead:   false,
		},
		{
			ID:       shared.NewUUIDv7(),
			UserID:   adminUserID,
			Title:    "Przypomnienie o wygasającym dokumencie",
			Content:  "Paszport wygasa za 30 dni. Złóż wniosek online, aby uniknąć opóźnień.",
			Priority: "warning",
			Category: "administrative",
			IsRead:   false,
		},

		// --- Jan Kowalski ---
		{
			ID:       shared.NewUUIDv7(),
			UserID:   janKowalski,
			Title:    "Witamy w aplikacji Obywatel Plus",
			Content:  "Twój profil obywatela został zweryfikowany. Możesz teraz załatwiać sprawy urzędowe online.",
			Priority: "info",
			Category: "system",
			IsRead:   true,
		},
		{
			ID:       shared.NewUUIDv7(),
			UserID:   janKowalski,
			Title:    "Nowa wiadomość od Urzędu Miasta",
			Content:  "Otrzymałeś odpowiedź na zapytanie w sprawie zameldowania na pobyt czasowy.",
			Priority: "info",
			Category: "messages",
			IsRead:   false,
		},
		{
			ID:       shared.NewUUIDv7(),
			UserID:   janKowalski,
			Title:    "Decyzja podatkowa gotowa",
			Content:  "Wystawiono wymiar podatku od nieruchomości za bieżący rok.",
			Priority: "warning",
			Category: "administrative",
			IsRead:   false,
		},

		// --- Anna Nowak ---
		{
			ID:       shared.NewUUIDv7(),
			UserID:   annaNowak,
			Title:    "Aktywacja konta zakończona sukcesem",
			Content:  "Witaj Anno! Twoja tożsamość cyfrowa została pomyślnie potwierdzona.",
			Priority: "success",
			Category: "system",
			IsRead:   true,
		},
		{
			ID:       shared.NewUUIDv7(),
			UserID:   annaNowak,
			Title:    "Dowód osobisty gotowy do odbioru",
			Content:  "Twój nowy dowód osobisty czeka na odbiór w Urzędzie Gminy.",
			Priority: "success",
			Category: "administrative",
			IsRead:   false,
		},
		{
			ID:       shared.NewUUIDv7(),
			UserID:   annaNowak,
			Title:    "Weryfikacja dwuetapowa (2FA)",
			Content:  "Zarejestrowano nowe logowanie do Twojego konta z zaufanego urządzenia.",
			Priority: "info",
			Category: "security",
			IsRead:   false,
		},
	}

	if err := db.Create(&notifications).Error; err != nil {
		return fmt.Errorf("failed to seed notifications: %w", err)
	}

	log.Info(fmt.Sprintf("[SEED] Zakończono zasiewanie bazy powiadomień. Dodano %d wpisów.", len(notifications)))
	return nil
}
