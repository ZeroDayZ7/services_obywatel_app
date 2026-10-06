package config

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/notification-service/internal/model"
	"gorm.io/gorm"
)

// Zsynchronizowane UUID użytkowników z config/seed_user_db.go
var (
	rootUserID     = uuid.MustParse("707a8869-6867-4601-9337-e23fcb51b0ad") // Root / Admin
	officerUserID  = uuid.MustParse("e1f2a3b4-5566-7788-9900-aabbccddeeff") // Urzędnik
	citizenUserID1 = uuid.MustParse("a2f6b8c9-1122-4a55-8822-b98765432101") // Anna Nowak (anna@plus.pl)
	citizenUserID2 = uuid.MustParse("c3d4e5f6-3344-5b66-9933-a12345678902") // Piotr Wiśniewski (piotr@plus.pl)
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

	log.Info("[SEED] Rozpoczynam zasiewanie bazy powiadomień dla wszystkich ról...")

	notifications := []model.Notification{
		// ==========================================
		// --- 1. ROOT / ADMIN (root@plus.pl) ---
		// ==========================================
		{
			ID:       shared.NewUUIDv7(),
			UserID:   rootUserID,
			Title:    "System Obywatel Plus gotowy do pracy",
			Content:  "Wszystkie mikroserwisy platformy działają prawidłowo. Infrastruktura K3s oraz węzły HSM są aktywne.",
			Priority: "info",
			Category: "system",
			IsRead:   true,
		},
		{
			ID:       shared.NewUUIDv7(),
			UserID:   rootUserID,
			Title:    "Alert Bezpieczeństwa: Weryfikacja Lynis",
			Content:  "Pomyślnie wykonano audyt bezpieczeństwa kontenera KMS. Brak krytycznych podatności.",
			Priority: "success",
			Category: "security",
			IsRead:   false,
		},
		{
			ID:       shared.NewUUIDv7(),
			UserID:   rootUserID,
			Title:    "Wymagana rotacja kluczy KMS",
			Content:  "Zbliża się cykliczny termin rotacji klucza głównego envelope encryption. Zaplanuj procedurę SSS.",
			Priority: "warning",
			Category: "security",
			IsRead:   false,
		},
		{
			ID:       shared.NewUUIDv7(),
			UserID:   rootUserID,
			Title:    "Wyszukiwanie Anomalii w SIEM (Wazuh)",
			Content:  "Wykryto 3 nieudane próby logowania na konto urzędnika z nieznanego adresu IP.",
			Priority: "error",
			Category: "security",
			IsRead:   false,
		},

		// ==========================================
		// --- 2. OFFICER / URZĘDNIK (officer@plus.pl) ---
		// ==========================================
		{
			ID:       shared.NewUUIDv7(),
			UserID:   officerUserID,
			Title:    "Nowe wnioski w kolejce urzędowej",
			Content:  "Własnie wpłynęły 4 nowe wnioski o wydanie dowodu osobistego wymagające Twojego zatwierdzenia.",
			Priority: "info",
			Category: "administrative",
			IsRead:   false,
		},
		{
			ID:       shared.NewUUIDv7(),
			UserID:   officerUserID,
			Title:    "Przydzielono zadanie: Podatek od nieruchomości",
			Content:  "Zostałeś wyznaczony jako osoba prowadząca sprawę nr URZ/2026/09/8821.",
			Priority: "info",
			Category: "administrative",
			IsRead:   true,
		},
		{
			ID:       shared.NewUUIDv7(),
			UserID:   officerUserID,
			Title:    "Weryfikacja tożsamości obywatela",
			Content:  "Wniosek użytkownika Piotr Wiśniewski oczekuje na weryfikację podpisu kwalifikowanego.",
			Priority: "warning",
			Category: "administrative",
			IsRead:   false,
		},

		// ==========================================
		// --- 3. CITIZEN 1: Anna Nowak (anna@plus.pl) ---
		// ==========================================
		{
			ID:       shared.NewUUIDv7(),
			UserID:   citizenUserID1,
			Title:    "Witamy w Obywatel Plus!",
			Content:  "Witaj Anno! Twoja tożsamość cyfrowa została pomyślnie potwierdzona profil zaufanym.",
			Priority: "success",
			Category: "system",
			IsRead:   true,
		},
		{
			ID:       shared.NewUUIDv7(),
			UserID:   citizenUserID1,
			Title:    "Dowód osobisty gotowy do odbioru",
			Content:  "Twój spersonalizowany spersonalizowany dowód osobisty czeka na odbiór w Urzędzie Gminy (Stanowisko 4).",
			Priority: "success",
			Category: "administrative",
			IsRead:   false,
		},
		{
			ID:       shared.NewUUIDv7(),
			UserID:   citizenUserID1,
			Title:    "Nowe logowanie do aplikacji",
			Content:  "Zarejestrowano pomyślne logowanie z nowego urządzenia mobilnego (Android / Tailscale mesh).",
			Priority: "info",
			Category: "security",
			IsRead:   true,
		},

		// ==========================================
		// --- 4. CITIZEN 2: Piotr Wiśniewski (piotr@plus.pl) ---
		// ==========================================
		{
			ID:       shared.NewUUIDv7(),
			UserID:   citizenUserID2,
			Title:    "Konto Obywatela aktywne",
			Content:  "Twój profil obywatelski uzyskał pełen poziom zweryfikowania tożsamości cyfrowej.",
			Priority: "info",
			Category: "system",
			IsRead:   true,
		},
		{
			ID:       shared.NewUUIDv7(),
			UserID:   citizenUserID2,
			Title:    "Decyzja podatkowa za rok 2026",
			Content:  "Wystawiono decyzję w sprawie wymiaru podatku od nieruchomości. Termin płatności: 14 dni.",
			Priority: "warning",
			Category: "payments",
			IsRead:   false,
		},
		{
			ID:       shared.NewUUIDv7(),
			UserID:   citizenUserID2,
			Title:    "Odpowiedź na wniosek zameldowania",
			Content:  "Urząd Miasta wydał pozytywną decyzję w sprawie zameldowania na pobyt czasowy.",
			Priority: "success",
			Category: "administrative",
			IsRead:   false,
		},
		{
			ID:       shared.NewUUIDv7(),
			UserID:   citizenUserID2,
			Title:    "Zaproszenie do e-Głosowania",
			Content:  "Otwarto lokalny konsultacyjny plebiscyt w ramach Budżetu Obywatelskiego. Oddaj swój głos online!",
			Priority: "info",
			Category: "administrative",
			IsRead:   false,
		},
		{
			ID:       shared.NewUUIDv7(),
			UserID:   citizenUserID2,
			Title:    "Przypomnienie: Wygasający przegląd pojazdu",
			Content:  "Badanie techniczne Twojego pojazdu wygasa za 7 dni. Pamiętaj o wizycie w Stacji Kontroli Pojazdów.",
			Priority: "warning",
			Category: "administrative",
			IsRead:   false,
		},
		{
			ID:       shared.NewUUIDv7(),
			UserID:   citizenUserID2,
			Title:    "Bezpieczeństwo: Aktualizacja kodu PIN",
			Content:  "Pomyślnie zmieniono kod PIN do aplikacji Obywatel Plus z zaufanego urządzenia.",
			Priority: "info",
			Category: "security",
			IsRead:   true,
		},
	}

	if err := db.Create(&notifications).Error; err != nil {
		return fmt.Errorf("failed to seed notifications: %w", err)
	}

	log.Info(fmt.Sprintf("[SEED] Zakończono zasiewanie bazy powiadomień. Dodano %d wpisów.", len(notifications)))
	return nil
}
