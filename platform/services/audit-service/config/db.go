package config

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
)

// MustInitDB przyjmuje ustandaryzowany DBConfig z pkg
func MustInitDB(cfg viper.DBConfig) (*pgxpool.Pool, func()) {
	log := shared.GetLogger()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Parsowanie DSN
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		panic(fmt.Errorf("unable to parse DSN: %w", err))
	}

	// 2. Mapowanie uniwersalnych ustawień na pgxpool
	poolConfig.MaxConns = int32(cfg.MaxOpenConns)
	// pgx używa MaxConnIdleTime zamiast MaxIdleConns (liczby połączeń)
	poolConfig.MaxConnIdleTime = 5 * time.Minute
	poolConfig.MaxConnLifetime = cfg.ConnMaxLifetime

	// 3. Utworzenie połączenia
	dbPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		panic(fmt.Errorf("unable to create connection pool: %w", err))
	}

	// 4. Ping
	if err := dbPool.Ping(ctx); err != nil {
		panic(fmt.Errorf("database ping failed: %w", err))
	}

	// 5. Automatyczna migracja tabeli audytowej (opcjonalnie, jeśli nie używasz goose/atlas)
	// if err := EnsureAuditTable(ctx, dbPool); err != nil {
	// 	log.ErrorObj("Failed to ensure audit table", err)
	// }

	log.Info("Successfully connected to PostgreSQL via pgxpool (SQLC ready)")

	// 5. Bezpieczny Cleanup
	cleanup := func() {
		if dbPool != nil {
			log.Info("Closing PostgreSQL connection pool...")
			dbPool.Close()
		}
	}

	return dbPool, cleanup
}
