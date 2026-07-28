package config

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zerodayz7/platform/pkg/database"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
)

func MustInitDB(cfg viper.DBConfig) (*pgxpool.Pool, func()) {
	// 1. Inicjalizacja puli pgxpool
	dbPool, closeDB := database.MustInitDB(cfg)

	// 2. Tworzenie schematu "audit"
	ctx := context.Background()
	if err := database.EnsureSchemasPgx(ctx, dbPool, "audit"); err != nil {
		panic(err)
	}

	// 3. Uruchomienie migracji Goose
	if err := RunMigrations(cfg.GetDSN()); err != nil {
		panic(err)
	}

	return dbPool, closeDB
}

func RunMigrations(dsn string) error {
	log := shared.GetLogger()
	log.Info("Running Goose migrations for audit-service...")

	return database.RunMigrationsGoose(dsn, "db/migrations")
}
