package config

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/zerodayz7/platform/pkg/permissions"
	"github.com/zerodayz7/platform/pkg/security"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/auth-service/internal/model"
	"gorm.io/gorm"
)

var (
	// Definicje ID powiązane 1:1 z innymi usługami w systemie
	testUserID1 = uuid.MustParse("707a8869-6867-4601-9337-e23fcb51b0ad") // Jan Kowalski (root@plus.pl)
	testUserID2 = uuid.MustParse("a2f6b8c9-1122-4a55-8822-b98765432101") // Anna Nowak (anna@plus.pl)
	testUserID3 = uuid.MustParse("c3d4e5f6-3344-5b66-9933-a12345678902") // Piotr Wiśniewski (piotr@plus.pl)
	adminUserID = uuid.MustParse("92b98b5a-d0c3-410f-828d-2b30a585dea6") // admin@plus.pl
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

	log.Info("[SEED] Rozpoczynam zasiewanie użytkowników w auth-service...")

	// Dynamiczne generowanie hasha dla hasła "Zaq1@wsx"
	rawPassword := []byte("Zaq1@wsx")
	defer clear(rawPassword)

	hashedPassword, err := security.HashPassword(rawPassword, nil)
	if err != nil {
		return fmt.Errorf("failed to hash seed password: %w", err)
	}

	// Standardowy zestaw uprawnień obywatelskich
	userStandardPermissions := pq.StringArray{
		permissions.ReportsView,
		permissions.MessagesRead,
		permissions.MessagesWrite,
		permissions.MessagingAccess,
		permissions.DocumentsRead,
		permissions.DocumentsWrite,
	}

	users := []model.User{
		// 1. Jan Kowalski (Root)
		{
			ID:       testUserID1,
			Username: "root@plus.pl",
			Email:    "root@plus.pl",
			Password: hashedPassword,
			Role:     model.RoleRoot,
			Permissions: pq.StringArray{
				permissions.SystemAdmin,
				permissions.SystemManage,
				permissions.UsersRead,
				permissions.UsersWrite,
				permissions.UsersDelete,
				permissions.MessagesRead,
				permissions.MessagesWrite,
				permissions.MessagingAccess,
				permissions.DocumentsRead,
				permissions.DocumentsWrite,
			},
			TwoFactorEnabled: true,
		},
		// 2. Admin Systemowy
		{
			ID:       adminUserID,
			Username: "admin@plus.pl",
			Email:    "admin@plus.pl",
			Password: hashedPassword,
			Role:     model.RoleAdmin,
			Permissions: pq.StringArray{
				permissions.UsersRead,
				permissions.UsersWrite,
				permissions.ReportsView,
				permissions.ReportsExport,
				permissions.MessagesRead,
				permissions.MessagesWrite,
				permissions.MessagingAccess,
			},
			TwoFactorEnabled: true,
		},
		// 3. Anna Nowak
		{
			ID:               testUserID2,
			Username:         "anna@plus.pl",
			Email:            "anna@plus.pl",
			Password:         hashedPassword,
			Role:             model.RoleUser,
			Permissions:      userStandardPermissions,
			TwoFactorEnabled: true,
		},
		// 4. Piotr Wiśniewski
		{
			ID:               testUserID3,
			Username:         "piotr@plus.pl",
			Email:            "piotr@plus.pl",
			Password:         hashedPassword,
			Role:             model.RoleUser,
			Permissions:      userStandardPermissions,
			TwoFactorEnabled: true,
		},
	}

	return db.Transaction(func(tx *gorm.DB) error {
		for _, u := range users {
			if err := tx.Create(&u).Error; err != nil {
				return fmt.Errorf("failed to seed user %s: %w", u.Username, err)
			}
			log.Info(fmt.Sprintf("[SEED] Utworzono konto użytkownika: %-18s | ID: %s", u.Username, u.ID))
		}
		return nil
	})
}
