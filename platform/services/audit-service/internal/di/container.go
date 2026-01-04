package di

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/audit-service/db/dbgen"
	"github.com/zerodayz7/platform/services/audit-service/internal/audit"
)

type Container struct {
	AuditHandler *audit.AuditHandler
	AuditWorker  *audit.AuditWorker
	Redis        *redis.Client
}

func NewContainer(dbPool *pgxpool.Pool, redisClient *redis.Client, logger *shared.Logger) *Container {
	// 1. sqlc
	queries := dbgen.New(dbPool)

	// 2. Serwis (przekazujemy logger)
	auditSvc := audit.NewAuditService(queries, logger)

	// 3. Handler
	auditH := audit.NewAuditHandler(auditSvc)

	// 4. Worker
	auditW := audit.NewAuditWorker(redisClient, auditSvc, logger)

	return &Container{
		AuditHandler: auditH,
		AuditWorker:  auditW,
		Redis:        redisClient,
	}
}
