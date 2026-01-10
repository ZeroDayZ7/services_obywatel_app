package database

import "gorm.io/gorm"

// Seeder to interfejs dla Twoich funkcji SeedData
type Seeder func(*gorm.DB) error

// RunSeed uruchamia seeder tylko jeśli baza spełnia warunek (np. brak rekordów w podanej tabeli)
func RunSeed(db *gorm.DB, model any, seeder Seeder) error {
	var count int64
	// Sprawdzamy czy tabela dla danego modelu jest pusta
	if err := db.Model(model).Count(&count).Error; err != nil {
		return err
	}

	if count == 0 {
		return seeder(db)
	}
	return nil
}
