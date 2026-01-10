package config

import (
	"github.com/zerodayz7/platform/pkg/database"
	"github.com/zerodayz7/platform/pkg/viper"
	"github.com/zerodayz7/platform/services/notification-service/internal/model"
	"gorm.io/gorm"
)

func MustInitDB(cfg viper.DBConfig) (*gorm.DB, func()) {
	// 1. Inicjalizacja z pkg - przekazujemy tylko model powiadomienia
	// pkg/database sam zadba o MaxOpenConns, Ping i całą resztę.
	db, closeDB, err := database.NewPostgres(cfg,
		&model.Notification{},
	)
	if err != nil {
		// Jeśli baza jest niezbędna do działania serwisu, panic jest tu ok
		panic(err)
	}

	// Jeśli nie masz SeedData dla notyfikacji, to po prostu zwracasz połączenie
	return db, closeDB
}
