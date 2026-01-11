package config

import (
	"github.com/zerodayz7/platform/pkg/database"
	"github.com/zerodayz7/platform/pkg/viper"
	"github.com/zerodayz7/platform/services/auth-service/internal/model"
	"gorm.io/gorm"
)

func MustInitDB(cfg viper.DBConfig) (*gorm.DB, func()) {
	// 1. Inicjalizacja z pkg - przekazujemy modele do migracji
	db, closeDB, err := database.NewPostgres(cfg,
		&model.User{},
		&model.UserPermission{},
		&model.RefreshToken{},
		&model.UserDevice{},
	)
	if err != nil {
		panic(err)
	}

	// 2. Uruchomienie seedera przy użyciu pomocnika z pkg
	// Podajemy model User, żeby sprawdzić czy tabela jest pusta przed seedem
	if err := database.RunSeed(db, &model.User{}, SeedData); err != nil {
		panic(err)
	}

	return db, closeDB
}
