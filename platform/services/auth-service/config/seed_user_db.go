package config

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/permissions"
	"github.com/zerodayz7/platform/pkg/security"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/auth-service/internal/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	// Statyczne ID użytkowników
	rootUserID     = uuid.MustParse("707a8869-6867-4601-9337-e23fcb51b0ad") // Root
	officerUserID  = uuid.MustParse("e1f2a3b4-5566-7788-9900-aabbccddeeff") // Urzędnik
	citizenUserID1 = uuid.MustParse("a2f6b8c9-1122-4a55-8822-b98765432101") // Anna Nowak
	citizenUserID2 = uuid.MustParse("c3d4e5f6-3344-5b66-9933-a12345678902") // Piotr Wiśniewski

	// Statyczne ID Organizacji / Departamentów
	testInstitutionID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	testDepartmentID  = uuid.MustParse("22222222-2222-2222-2222-222222222222")
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

	log.Info("[SEED] Rozpoczynam zasiewanie użytkowników i profilów pracowniczych...")

	rawPassword := []byte("Zaq1@wsx")
	defer clear(rawPassword)

	hashedPassword, err := security.HashPassword(rawPassword, nil)
	if err != nil {
		return fmt.Errorf("failed to hash seed password: %w", err)
	}

	// 1. Definicja kont tożsamości (User)
	users := []model.User{
		{
			ID:               rootUserID,
			Username:         "root@plus.pl",
			Email:            "root@plus.pl",
			Password:         hashedPassword,
			Role:             model.RoleRoot,
			TwoFactorEnabled: true,
		},
		{
			ID:               officerUserID,
			Username:         "urzednik@plus.pl",
			Email:            "urzednik@plus.pl",
			Password:         hashedPassword,
			Role:             model.RoleOfficer,
			TwoFactorEnabled: true,
		},
		{
			ID:               citizenUserID1,
			Username:         "anna@plus.pl",
			Email:            "anna@plus.pl",
			Password:         hashedPassword,
			Role:             model.RoleCitizen,
			TwoFactorEnabled: true,
		},
		{
			ID:               citizenUserID2,
			Username:         "piotr@plus.pl",
			Email:            "piotr@plus.pl",
			Password:         hashedPassword,
			Role:             model.RoleCitizen,
			TwoFactorEnabled: true,
		},
	}

	// 2. Uprawnienia pracownicze dla Urzędnika
	officerPermissions := datatypes.JSONSlice[string]{
		permissions.UsersRead,
		permissions.UsersWrite,
		permissions.MessagesRead,
		permissions.MessagesWrite,
		permissions.MessagingAccess,
		permissions.DocumentsRead,
		permissions.DocumentsWrite,
		permissions.ReportsView,
		permissions.ReportsExport,
	}

	// 3. Profil pracowniczy powiązany z kontem Urzędnika
	employeeProfile := model.EmployeeProfile{
		UserID:         officerUserID,
		EmployeeNumber: "EMP-2026-0001",
		InstitutionID:  testInstitutionID,
		DepartmentID:   testDepartmentID,
		Permissions:    officerPermissions,
		Active:         true,
	}

	return db.Transaction(func(tx *gorm.DB) error {
		// Tworzenie kont użytkowników
		for _, u := range users {
			if err := tx.Create(&u).Error; err != nil {
				return fmt.Errorf("failed to seed user %s: %w", u.Username, err)
			}
			log.Info(fmt.Sprintf("[SEED] Utworzono użytkownika: %-18s | ID: %s | Role: %s", u.Username, u.ID, u.Role))
		}

		// Tworzenie profilu urzędnika
		if err := tx.Create(&employeeProfile).Error; err != nil {
			return fmt.Errorf("failed to seed employee profile: %w", err)
		}
		log.Info(fmt.Sprintf("[SEED] Utworzono profil pracownika dla ID: %s", employeeProfile.UserID))

		return nil
	})
}
