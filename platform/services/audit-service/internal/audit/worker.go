package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

func (w *AuditWorker) Start(ctx context.Context) {
	if w.redis == nil {
		w.logger.Error("Audit Worker: Redis client is nil, terminating worker")
		return
	}

	if err := w.ensureRedisInfrastructure(ctx); err != nil {
		w.logger.ErrorObj("Worker: failed to bootstrap redis infra", err)
		return
	}

	w.logger.Info("Audit Worker: Listening for events...")

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Audit Worker: Stopping gracefully...")
			return
		default:
			w.processBatch(ctx)
		}
	}
}

func (w *AuditWorker) processBatch(ctx context.Context) {
	entries, err := w.redis.ReadStreamBatch(
		ctx,
		auditStream,
		auditGroup,
		auditConsumer,
		batchSize,
		batchTimeout,
	)
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return
		}
		if strings.Contains(err.Error(), "NOGROUP") {
			w.logger.Warn("Worker: consumer group missing, recreating...")
			_ = w.ensureRedisInfrastructure(ctx)
			time.Sleep(1 * time.Second)
			return
		}
		w.logger.ErrorObj("Worker: Redis read error", err)
		time.Sleep(2 * time.Second)
		return
	}

	if len(entries) == 0 {
		return
	}

	batch := make([]AuditMessage, 0, len(entries))
	validACKIDs := make([]string, 0, len(entries))
	poisonACKIDs := make([]string, 0)

	for _, entry := range entries {
		rawPayload, ok := entry.Values["payload"].(string)
		if !ok {
			w.logger.WarnMap("Worker: invalid/missing payload schema", map[string]any{"entry_id": entry.ID})
			poisonACKIDs = append(poisonACKIDs, entry.ID)
			continue
		}

		var msg AuditMessage
		if err := json.Unmarshal([]byte(rawPayload), &msg); err != nil {
			w.logger.ErrorObj("Worker: JSON unmarshal failed", fmt.Errorf("id %s: %w", entry.ID, err))
			poisonACKIDs = append(poisonACKIDs, entry.ID)
			continue
		}

		batch = append(batch, msg)
		validACKIDs = append(validACKIDs, entry.ID)
	}

	// Odrzucenie uszkodzonych wiadomości (Poison Pill Guard) - ackujemy, żeby nie blokowały grupy
	if len(poisonACKIDs) > 0 {
		_ = w.redis.AckStreamBatch(ctx, auditStream, auditGroup, poisonACKIDs)
	}

	if len(batch) == 0 {
		return
	}

	// Persystencja do bazy danych
	if err := w.svc.SaveLogsBatch(ctx, batch); err != nil {
		w.logger.ErrorObj("Worker: SaveLogsBatch failed - batch deferred to PEL", err)
		// Brak ACK: dane pozostają w Redis PEL i zostaną przetworzone po restarcie / ponowieniu
		time.Sleep(1 * time.Second)
		return
	}

	// ACK przetworzonego batcha
	if err := w.redis.AckStreamBatch(ctx, auditStream, auditGroup, validACKIDs); err != nil {
		w.logger.ErrorObj("Worker: AckStreamBatch failed", err)
		return
	}

	w.logger.InfoMap("Worker: Batch processed successfully", map[string]any{
		"processed_count": len(batch),
		"poison_count":    len(poisonACKIDs),
	})
}

// =======================================================
// 🔧 INFRA SELF-HEALING
// =======================================================

func (w *AuditWorker) ensureRedisInfrastructure(ctx context.Context) error {
	// Rezerwacja i tworzenie grupy przy użyciu natywnego MKSTREAM (bez zanieczyszczania bazy sztucznym rekordem)
	if err := w.redis.EnsureGroup(ctx, auditStream, auditGroup); err != nil {
		return fmt.Errorf("failed to ensure redis stream group: %w", err)
	}
	return nil
}
