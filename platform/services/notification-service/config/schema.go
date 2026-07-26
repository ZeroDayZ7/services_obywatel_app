package config

import (
	"fmt"

	"gorm.io/gorm"
)

const DefaultSchema = "public"

func AllSchemas() []string {
	return []string{}
}

func EnsureSchemas(db *gorm.DB) error {
	for _, schema := range AllSchemas() {
		if schema == "public" {
			continue
		}
		query := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s;", schema)
		if err := db.Exec(query).Error; err != nil {
			return fmt.Errorf("failed to ensure schema %s: %w", schema, err)
		}
	}
	return nil
}
