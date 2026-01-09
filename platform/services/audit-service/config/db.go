package config

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zerodayz7/platform/pkg/shared"
)

func MustInitDB(cfg DBConfig) (*pgxpool.Pool, func()) {
	log := shared.GetLogger()
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

func EnsureAuditTable(ctx context.Context, db *pgxpool.Pool) error {
	_, err := db.Exec(ctx, `
	-- Usuń starą tabelę, jeśli istnieje
	DROP TABLE IF EXISTS audit_logs;

	-- Tworzymy nową tabelę
	CREATE TABLE audit_logs (
		id            BIGSERIAL PRIMARY KEY,
		user_id       UUID NOT NULL,
		service_name  VARCHAR(50) NOT NULL,
		action        VARCHAR(100) NOT NULL,
		ip_address    VARCHAR(45) NOT NULL,
		metadata      JSONB NOT NULL,
		status        VARCHAR(20) NOT NULL,
		created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	-- Indeksy dla wydajności
	CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_logs(user_id);
	CREATE INDEX IF NOT EXISTS idx_audit_ip ON audit_logs(ip_address);
	CREATE INDEX IF NOT EXISTS idx_audit_service ON audit_logs(service_name);
	`)
	return err
}
