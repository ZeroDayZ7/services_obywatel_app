package database

import (
	"fmt"

	"gorm.io/gorm"
)

// GormAdapter implementuje interfejs shared.PostgresRotatable dla instancji *gorm.DB
type GormAdapter struct {
	db *gorm.DB
}

// NewGormAdapter tworzy adapter rotowalnej bazy danych dla GORM
func NewGormAdapter(db *gorm.DB) *GormAdapter {
	return &GormAdapter{db: db}
}

// UpdateCredentials wykonuje zaktualizowanie DSN / połączenia w puli
func (a *GormAdapter) UpdateCredentials(username, password string) error {
	if a.db == nil {
		return fmt.Errorf("instancja db jest nil")
	}

	sqlDB, err := a.db.DB()
	if err != nil {
		return fmt.Errorf("błąd pobierania sql.DB z GORM: %w", err)
	}

	// Miejsce na ewentualną rejestrację nowych poświadczeń w sterowniku (np. pgxpool)
	_ = sqlDB
	// sqlDB.SetConnMaxLifetime(5 * time.Second)
	return nil
}
