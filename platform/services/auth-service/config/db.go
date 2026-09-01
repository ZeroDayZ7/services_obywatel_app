package config

import (
	"github.com/zerodayz7/platform/pkg/database"
	"github.com/zerodayz7/platform/pkg/viper"
	"gorm.io/gorm"
)

//#region MustInitDB
func MustInitDB(cfg viper.DBConfig) (*gorm.DB, func()) {
	// 1. Inicjalizacja połączenia
	db, closeDB, err := database.NewPostgres(cfg)
	if err != nil {
		panic(err)
	}

	// 2. Automatyczne upewnienie się, że schematy istnieją (z wykorzystaniem pkg/database)
	if err := database.EnsureSchemas(db, AllSchemas()...); err != nil {
		panic(err)
	}

	// // 3. Wykonanie AutoMigrate dla modeli
	if err := AutoMigrate(db); err != nil {
		panic(err)
	}

	// // 4. Uruchomienie seedera
	if err := SeedData(db); err != nil {
		panic(err)
	}

	return db, closeDB
}
