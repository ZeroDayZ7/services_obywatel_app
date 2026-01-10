package config

import (
	"github.com/zerodayz7/platform/pkg/database"
	"github.com/zerodayz7/platform/pkg/viper"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/model"
	"gorm.io/gorm"
)

// MustInitDB inicjalizuje połączenie z bazą danych PostgreSQL oraz wykonuje migracje modeli.
func MustInitDB(cfg viper.DBConfig) (*gorm.DB, func()) {
	// 1. Inicjalizacja połączenia i automatyczna migracja schematów dokumentów
	db, closeDB, err := database.NewPostgres(cfg,
		&model.CitizenProfile{},
		&model.UserDocument{},
	)
	if err != nil {
		panic(err)
	}

	// 2. Opcjonalne uruchomienie seedera (jeśli zdefiniowano SeedData)
	// if err := database.RunSeed(db, &model.CitizenProfile{}, SeedData); err != nil {
	// 	panic(err)
	// }

	return db, closeDB
}
