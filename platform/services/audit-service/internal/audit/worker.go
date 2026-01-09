package audit

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/shared"
)

const (
	auditStream   = "audit_stream"
	auditGroup    = "audit_service_group"
	auditConsumer = "worker_1"
	batchSize     = 100
	batchTimeout  = 500 * time.Millisecond
)

type AuditWorker struct {
	redis  *redis.Client
	svc    *AuditService
	logger *shared.Logger
}

func NewAuditWorker(r *redis.Client, s *AuditService, l *shared.Logger) *AuditWorker {
	return &AuditWorker{
		redis:  r,
		svc:    s,
		logger: l,
	}
}

func (w *AuditWorker) Start() {
	ctx := context.Background()

	if err := w.ensureRedisInfrastructure(ctx); err != nil {
		w.logger.ErrorObj("Worker: failed to bootstrap redis infra", err)
		return
	}

	w.logger.Info("Audit Worker: Listening for events...")

	for {
		entries, err := w.redis.ReadStreamBatch(
			ctx,
			auditStream,
			auditGroup,
			auditConsumer,
			batchSize,
			batchTimeout,
		)
		if err != nil {
			if strings.Contains(err.Error(), "NOGROUP") {
				w.logger.Warn("Worker: consumer group missing, recreating...")
				if err := w.ensureRedisInfrastructure(ctx); err != nil {
					w.logger.ErrorObj("Worker: failed to recreate redis infra", err)
					time.Sleep(5 * time.Second)
				}
				continue
			}
			w.logger.ErrorObj("Worker: Redis error", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if len(entries) == 0 {
			continue
		}

		var batch []AuditMessage
		var ackIDs []string

		for _, entry := range entries {
			rawPayload, ok := entry.Values["payload"].(string)
			if !ok {
				w.logger.WarnMap("Worker: payload missing or invalid", map[string]any{
					"entry_id": entry.ID,
				})
				continue
			}

			var msg AuditMessage
			if err := json.Unmarshal([]byte(rawPayload), &msg); err != nil {
				w.logger.ErrorObj("Worker: JSON unmarshal failed", err)
				continue
			}

			batch = append(batch, msg)
			ackIDs = append(ackIDs, entry.ID)
		}

		if len(batch) == 0 {
			continue
		}

		// 🔥 Zapis batchem do DB
		if err := w.svc.SaveLogsBatch(ctx, batch); err != nil {
			w.logger.ErrorObj("Worker: SaveLogsBatch failed", err)
			continue
		}

		// 🔥 ACK batchem w Redisie
		if err := w.redis.AckStreamBatch(ctx, auditStream, auditGroup, ackIDs); err != nil {
			w.logger.ErrorObj("Worker: AckStreamBatch failed", err)
			continue
		}

		w.logger.InfoMap("Worker: Batch processed", map[string]any{
			"batch_count": len(batch),
			"ack_count":   len(ackIDs),
		})
	}
}

// =======================================================
// 🔧 INFRA SELF-HEALING
// =======================================================

func (w *AuditWorker) ensureRedisInfrastructure(ctx context.Context) error {
	// Wymuś istnienie streama (XADD noop)
	if err := w.redis.SendAuditLog(ctx, auditStream, map[string]any{
		"_bootstrap": true,
		"_ts":        time.Now().Unix(),
	}); err != nil {
		return err
	}

	// Zapewnij istnienie consumer group
	if err := w.redis.EnsureGroup(ctx, auditStream, auditGroup); err != nil {
		return err
	}

	return nil
}
