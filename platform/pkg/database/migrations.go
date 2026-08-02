package database

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/zerodayz7/platform/pkg/shared"
)

func RunMigrationsGoose(dsn string, migrationsDir string) error {
	log := shared.GetLogger()

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("failed to open db for migrations: %w", err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	log.Info("Running Goose migrations...")
	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("goose up failed: %w", err)
	}

	return nil
}
