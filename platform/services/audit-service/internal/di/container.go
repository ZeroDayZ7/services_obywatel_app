package di

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
	"github.com/zerodayz7/platform/services/audit-service/db/dbgen"
	"github.com/zerodayz7/platform/services/audit-service/internal/audit"
)

type Container struct {
	AuditHandler *audit.AuditHandler
	AuditWorker  *audit.AuditWorker
	Redis        *redis.Client
	Logger       *shared.Logger
	Config       *viper.Config
}

// NewContainer teraz przyjmuje cfg, aby wstrzyknąć go do aplikacji i handlerów.
func NewContainer(dbPool *pgxpool.Pool, redisClient *redis.Client, logger *shared.Logger, cfg *viper.Config) *Container {
	// 1. Inicjalizacja zapytań sqlc
	queries := dbgen.New(dbPool)

	// 2. Serwis (logika biznesowa)
	auditSvc := audit.NewAuditService(queries, logger)

	// 3. Handler (warstwa HTTP)
	auditH := audit.NewAuditHandler(auditSvc, logger)

	// 4. Worker (procesy asynchroniczne Redis)
	auditW := audit.NewAuditWorker(redisClient, auditSvc, logger)

	return &Container{
		AuditHandler: auditH,
		AuditWorker:  auditW,
		Redis:        redisClient,
		Logger:       logger,
		Config:       cfg, // Mapowanie przekazanego configu
	}
}
