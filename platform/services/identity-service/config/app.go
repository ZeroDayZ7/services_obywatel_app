package config

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	Config *Config
	DB     *pgxpool.Pool
}

func InitApp() (*App, func()) {
	if err := LoadConfigGlobal(); err != nil {
		panic(err)
	}

	dbPool, closeDB := MustInitDB(AppConfig.Database)

	app := &App{
		Config: &AppConfig,
		DB:     dbPool,
	}

	return app, closeDB
}