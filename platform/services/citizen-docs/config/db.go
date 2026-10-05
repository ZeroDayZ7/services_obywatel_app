package config

import (
	"github.com/zerodayz7/platform/pkg/database"
	"github.com/zerodayz7/platform/pkg/viper"
	"gorm.io/gorm"
)

func MustInitDB(cfg viper.DBConfig) (*gorm.DB, func()) {
	db, closeDB, err := database.NewPostgres(cfg)
	if err != nil {
		panic(err)
	}

	if err := database.EnsureSchemas(db, AllSchemas()...); err != nil {
		panic(err)
	}

	if err := AutoMigrate(db); err != nil {
		panic(err)
	}

	if err := SeedData(db); err != nil {
		panic(err)
	}

	return db, closeDB
}
