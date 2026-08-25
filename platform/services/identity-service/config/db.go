package config

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zerodayz7/platform/pkg/database"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
)

//#region MustInitDB
func MustInitDB(cfg viper.DBConfig) (*pgxpool.Pool, func()) {
	dbPool, closeDB := database.MustInitDB(cfg)

	if err := RunMigrations(cfg.GetDSN()); err != nil {
		panic(err)
	}

	return dbPool, closeDB
}

//#region RunMigrations
func RunMigrations(dsn string) error {
	log := shared.GetLogger()
	log.Info("Running Goose migrations for citizen-service...")

	return database.RunMigrationsGoose(dsn, "db/migrations")
}
