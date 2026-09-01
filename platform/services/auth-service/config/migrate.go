package config

import (
	"fmt"

	"github.com/zerodayz7/platform/services/auth-service/internal/model"
	"gorm.io/gorm"
)

// #region AutoMigrate
func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&model.User{},
		&model.RefreshToken{},
		&model.UserDevice{},
		&model.EmployeeProfile{},
		&model.EmployeeCredential{},
		&model.AvailablePermission{},
	); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}
