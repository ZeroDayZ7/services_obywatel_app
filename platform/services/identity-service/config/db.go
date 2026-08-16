package config

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zerodayz7/platform/pkg/database"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
)

func MustInitDB(cfg viper.DBConfig) (*pgxpool.Pool, func()) {
	dbPool, closeDB := database.MustInitDB(cfg)

	ctx := context.Background()
	if err := database.EnsureSchemasPgx(ctx, dbPool, "citizens"); err != nil {
		panic(err)
	}

	if err := RunMigrations(cfg.GetDSN()); err != nil {
		panic(err)
	}

	return dbPool, closeDB
}

func RunMigrations(dsn string) error {
	log := shared.GetLogger()
	log.Info("Running Goose migrations for citizen-service...")

	return database.RunMigrationsGoose(dsn, "db/migrations")
}