package config

import (
	"fmt"
	"time"

	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/notification-service/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// MustInitDB inicjalizuje połączenie z PostgreSQL i wykonuje migracje
func MustInitDB(cfg DBConfig) (*gorm.DB, func()) {
	log := shared.GetLogger()

	db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{})
	if err != nil {
		panic(fmt.Errorf("failed to connect to database: %w", err))
	}

	sqlDB, err := db.DB()
	if err != nil {
		panic(fmt.Errorf("failed to get database instance: %w", err))
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		panic(fmt.Errorf("database ping failed: %w", err))
	}

	// Migracje modeli Notification Service
	if err := db.AutoMigrate(
		&model.Notification{},
	); err != nil {
		log.ErrorObj("Failed to migrate database", err)
		panic(err)
	}

	log.Info("Successfully connected to PostgreSQL")
	return db, func() { sqlDB.Close() }
}
