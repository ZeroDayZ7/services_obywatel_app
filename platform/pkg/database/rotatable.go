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
func (a *GormAdapter) UpdateCredentials(username string, password []byte) error {
	if a.db == nil {
		return fmt.Errorf("instancja db jest nil")
	}

	sqlDB, err := a.db.DB()
	if err != nil {
		return fmt.Errorf("błąd pobierania sql.DB z GORM: %w", err)
	}

	// TODO: Konfiguracja re-dialera / podmienienie poświadczeń w sterowniku (np. pgxpool.Config / Driver)
	_ = sqlDB
	_ = username

	return nil
}
