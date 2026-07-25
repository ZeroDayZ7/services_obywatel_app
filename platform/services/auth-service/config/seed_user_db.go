package config

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/auth-service/internal/model"
	"gorm.io/gorm"
)

func SeedUsers(db *gorm.DB) error {
	log := shared.GetLogger()

	var count int64
	if err := db.Model(&model.User{}).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to count users: %w", err)
	}

	if count > 0 {
		return nil
	}

	log.Info("[SEED] Rozpoczynam zasiewanie użytkowników, umów i kodów PUK...")

	// Hasło testowe: Zaq1@wsx
	testPassword := "WmixUVuBuUWGZzbV2lmXrA$0okaGZfyu+EJgNRSI6aSIyB+WvMDFiBKyN0P+DW7294"

	// Hash kodu PUK "12345678" dla środowiska lokalnego (bcrypt/argon)
	testPukHash := "$2a$10$wN10O2A1RzE1.p.2N9wP5O8Z5.M4G.q/m8YyYgX4Xv/Nq5UuS3x.S"

	usersData := []struct {
		User      model.User
		Agreement model.UserAgreement
		PukCode   model.UserPukCode
	}{
		{
			User: model.User{
				ID:               uuid.New(),
				Username:         "root@plus.pl",
				Email:            "root@plus.pl",
				Password:         testPassword,
				Role:             model.RoleRoot,
				Permissions:      pq.StringArray{model.PermSystemAdmin, model.PermSystemManage, model.PermUsersRead, model.PermUsersWrite, model.PermUsersDelete},
				TwoFactorEnabled: true,
			},
			Agreement: model.UserAgreement{
				AgreementNumber: "UM/2026/01/0001",
				PeselEncrypted:  "ENC_PESEL_ROOT_89010112345",
				VerifiedPhone:   "+48500000001",
				Status:          model.AgreementStatusActive,
				SignedAt:        time.Now().AddDate(-1, 0, 0),
				VerifiedVia:     "BRANCH",
			},
			PukCode: model.UserPukCode{
				PukHash:     testPukHash,
				Status:      model.PukStatusActive,
				MaxAttempts: 3,
			},
		},
		{
			User: model.User{
				ID:               uuid.New(),
				Username:         "admin@plus.pl",
				Email:            "admin@plus.pl",
				Password:         testPassword,
				Role:             model.RoleAdmin,
				Permissions:      pq.StringArray{model.PermUsersRead, model.PermUsersWrite, model.PermReportsView, model.PermReportsExport},
				TwoFactorEnabled: true,
			},
			Agreement: model.UserAgreement{
				AgreementNumber: "UM/2026/01/0002",
				PeselEncrypted:  "ENC_PESEL_ADMIN_90020223456",
				VerifiedPhone:   "+48500000002",
				Status:          model.AgreementStatusActive,
				SignedAt:        time.Now().AddDate(-1, 0, 0),
				VerifiedVia:     "BRANCH",
			},
			PukCode: model.UserPukCode{
				PukHash:     testPukHash,
				Status:      model.PukStatusActive,
				MaxAttempts: 3,
			},
		},
		{
			User: model.User{
				ID:               uuid.New(),
				Username:         "user@example.com",
				Email:            "user@example.com",
				Password:         testPassword,
				Role:             model.RoleUser,
				Permissions:      pq.StringArray{model.PermReportsView},
				TwoFactorEnabled: true,
			},
			Agreement: model.UserAgreement{
				AgreementNumber: "UM/2026/01/0003",
				PeselEncrypted:  "ENC_PESEL_USER_95030334567",
				VerifiedPhone:   "+48500000003",
				Status:          model.AgreementStatusActive,
				SignedAt:        time.Now().AddDate(-1, 0, 0),
				VerifiedVia:     "MOJE_ID",
			},
			PukCode: model.UserPukCode{
				PukHash:     testPukHash,
				Status:      model.PukStatusActive,
				MaxAttempts: 3,
			},
		},
	}

	return db.Transaction(func(tx *gorm.DB) error {
		for _, item := range usersData {
			// 1. Zapis użytkownika
			if err := tx.Create(&item.User).Error; err != nil {
				return fmt.Errorf("failed to seed user %s: %w", item.User.Username, err)
			}

			// 2. Zapis umowy z powiązanym UserID
			item.Agreement.ID = uuid.New()
			item.Agreement.UserID = item.User.ID
			if err := tx.Create(&item.Agreement).Error; err != nil {
				return fmt.Errorf("failed to seed agreement for user %s: %w", item.User.Username, err)
			}

			// 3. Zapis PUK z powiązanym UserID i UserAgreementID
			item.PukCode.ID = uuid.New()
			item.PukCode.UserID = item.User.ID
			item.PukCode.UserAgreementID = item.Agreement.ID
			if err := tx.Create(&item.PukCode).Error; err != nil {
				return fmt.Errorf("failed to seed puk code for user %s: %w", item.User.Username, err)
			}

			log.Info(fmt.Sprintf("[SEED] Utworzono użytkownika: %-18s | Umowa: %s | PUK: AKTYWNY", item.User.Username, item.Agreement.AgreementNumber))
		}
		return nil
	})
}
