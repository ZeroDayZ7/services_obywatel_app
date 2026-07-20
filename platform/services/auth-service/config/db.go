package config

import (
	"fmt"

	"github.com/zerodayz7/platform/pkg/database"
	"github.com/zerodayz7/platform/pkg/viper"
	"github.com/zerodayz7/platform/services/auth-service/internal/model"
	"gorm.io/gorm"
)

func ensureSchemas(db *gorm.DB) error {
	for _, s := range AllSchemas() {
		query := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s;", s)
		if err := db.Exec(query).Error; err != nil {
			return fmt.Errorf("failed to ensure schema %s: %w", s, err)
		}
	}
	return nil
}

func MustInitDB(cfg viper.DBConfig) (*gorm.DB, func()) {
	// 1. Inicjalizacja połączenia bez przekazywania modeli (sam klient SQL)
	db, closeDB, err := database.NewPostgres(cfg)
	if err != nil {
		panic(err)
	}

	// 2. Automatyczne upewnienie się, że schematy istnieją
	if err := ensureSchemas(db); err != nil {
		panic(err)
	}

	// 3. Wykonanie AutoMigrate dla modeli
	if err := db.AutoMigrate(
		&model.User{},
		&model.RefreshToken{},
		&model.UserDevice{},
	); err != nil {
		panic(fmt.Sprintf("failed to run migrations: %v", err))
	}

	// 4. Uruchomienie seedera przy użyciu pomocnika z pkg
	if err := database.RunSeed(db, &model.User{}, SeedData); err != nil {
		panic(err)
	}

	return db, closeDB
}
