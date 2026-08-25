package config

import (
	"fmt"

	"gorm.io/gorm"
)

//#region SeedData
func SeedData(db *gorm.DB) error {
	if err := SeedPermissions(db); err != nil {
		return fmt.Errorf("permission seeding failed: %w", err)
	}

	if err := SeedUsers(db); err != nil {
		return fmt.Errorf("user seeding failed: %w", err)
	}

	if err := SeedInitialEmployeeCredential(db); err != nil {
		return fmt.Errorf("user seeding failed: %w", err)
	}

	return nil
}
