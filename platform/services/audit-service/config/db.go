package config

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
)

func MustInitDB(cfg viper.DBConfig) (*pgxpool.Pool, func()) {
	log := shared.GetLogger()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(cfg.GetDSN())
	if err != nil {
		panic(fmt.Errorf("unable to parse DSN: %w", err))
	}

	poolConfig.MaxConns = int32(cfg.MaxOpenConns)
	poolConfig.MaxConnIdleTime = 5 * time.Minute
	poolConfig.MaxConnLifetime = cfg.ConnMaxLifetime

	dbPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		panic(fmt.Errorf("unable to create connection pool: %w", err))
	}

	if err := dbPool.Ping(ctx); err != nil {
		panic(fmt.Errorf("database ping failed: %w", err))
	}

	log.Info("Successfully connected to PostgreSQL via pgxpool (SQLC ready)")

	cleanup := func() {
		if dbPool != nil {
			log.Info("Closing PostgreSQL connection pool...")
			dbPool.Close()
		}
	}

	return dbPool, cleanup
}
