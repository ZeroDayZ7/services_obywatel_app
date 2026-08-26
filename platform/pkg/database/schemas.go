package database

import (
	"fmt"

	"gorm.io/gorm"
)

// EnsureSchemas creates provided schemas in PostgreSQL if they don't already exist.
// Ignores empty schema names or "public" as it always exists by default.
//#region EnsureSchemas
func EnsureSchemas(db *gorm.DB, schemas ...string) error {
	for _, schema := range schemas {
		if schema == "" || schema == "public" {
			continue
		}

		query := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s;", schema)
		if err := db.Exec(query).Error; err != nil {
			return fmt.Errorf("failed to ensure schema %s: %w", schema, err)
		}
	}

	return nil
}
