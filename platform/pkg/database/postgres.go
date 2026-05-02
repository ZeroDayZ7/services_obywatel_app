package database

import (
	"fmt"

	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewPostgres(cfg viper.DBConfig, models ...any) (*gorm.DB, func(), error) {
	log := shared.GetLogger()

	db, err := gorm.Open(postgres.Open(cfg.GetDSN()), &gorm.Config{
		PrepareStmt: true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	if err := sqlDB.Ping(); err != nil {
		return nil, nil, fmt.Errorf("database ping failed: %w", err)
	}

	if len(models) > 0 {
		log.Info("Running database migrations...")
		if err := db.AutoMigrate(models...); err != nil {
			return nil, nil, fmt.Errorf("migration failed: %w", err)
		}
	}

	closeDB := func() {
		if sqlDB != nil {
			log.Info("Closing PostgreSQL connection pool...")
			_ = sqlDB.Close()
		}
	}

	return db, closeDB, nil
}
