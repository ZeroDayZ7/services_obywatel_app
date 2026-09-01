package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
)

//#region NewPgxPool
// NewPgxPool tworzy pulę połączeń pgxpool bez zatrzymywania aplikacji przy błędzie.
func NewPgxPool(cfg viper.DBConfig) (*pgxpool.Pool, func(), error) {
	log := shared.GetLogger()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(cfg.GetDSN())
	if err != nil {
		return nil, nil, fmt.Errorf("unable to parse DSN: %w", err)
	}

	if cfg.MaxOpenConns > 0 {
		poolConfig.MaxConns = int32(cfg.MaxOpenConns)
	}
	poolConfig.MaxConnIdleTime = 5 * time.Minute
	poolConfig.MaxConnLifetime = cfg.ConnMaxLifetime

	dbPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := dbPool.Ping(ctx); err != nil {
		dbPool.Close()
		return nil, nil, fmt.Errorf("database ping failed: %w", err)
	}

	log.Info("Successfully connected to PostgreSQL via pgxpool (SQLC ready)")

	cleanup := func() {
		if dbPool != nil {
			log.Info("Closing PostgreSQL pgxpool connection pool...")
			dbPool.Close()
		}
	}

	return dbPool, cleanup, nil
}

//#region MustInitDB
// MustInitDB to wrapper panikujący na starcie (bootstrapper dla pgxpool).
func MustInitDB(cfg viper.DBConfig) (*pgxpool.Pool, func()) {
	pool, cleanup, err := NewPgxPool(cfg)
	if err != nil {
		panic(err)
	}
	return pool, cleanup
}

//#region EnsureSchemasPgx
// EnsureSchemasPgx tworzy schematy w PostgreSQL dla połączenia pgxpool.
func EnsureSchemasPgx(ctx context.Context, pool *pgxpool.Pool, schemas ...string) error {
	for _, schema := range schemas {
		if schema == "" || schema == "public" {
			continue
		}

		query := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s;", schema)
		if _, err := pool.Exec(ctx, query); err != nil {
			return fmt.Errorf("failed to ensure schema %s via pgxpool: %w", schema, err)
		}
	}

	return nil
}
