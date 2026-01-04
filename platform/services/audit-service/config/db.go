package config

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zerodayz7/platform/pkg/shared"
)

// Zwracamy *pgxpool.Pool zamiast *gorm.DB
func MustInitDB() (*pgxpool.Pool, func()) {
	log := shared.GetLogger()
	cfg := AppConfig.Database
	ctx := context.Background()

	// 1. Konfiguracja poola połączeń
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		panic(fmt.Errorf("unable to parse DSN: %w", err))
	}

	// Ustawienia wydajnościowe (pobrane z Twojego AppConfig)
	poolConfig.MaxConns = int32(cfg.MaxOpenConns)
	poolConfig.MaxConnIdleTime = time.Duration(cfg.MaxIdleConns) * time.Minute
	poolConfig.MaxConnLifetime = time.Duration(cfg.ConnMaxLifetime) * time.Minute

	// 2. Utworzenie połączenia
	dbPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		panic(fmt.Errorf("unable to create connection pool: %w", err))
	}

	// 3. Sprawdzenie połączenia (Ping)
	if err := dbPool.Ping(ctx); err != nil {
		panic(fmt.Errorf("database ping failed: %w", err))
	}

	log.Info("Successfully connected to PostgreSQL (via pgxpool)")

	// Funkcja zamykająca
	return dbPool, func() {
		dbPool.Close()
	}
}
