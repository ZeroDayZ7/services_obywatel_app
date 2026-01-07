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

	// 🔐 BOOTSTRAP: zapewnij istnienie grupy (i streama)
	if err := w.ensureRedisInfrastructure(ctx); err != nil {
		w.logger.ErrorObj("Worker: failed to bootstrap redis infra", err)
		return
	}

	w.logger.Info("Audit Worker: Listening for events...")

	for {
		entries, err := w.redis.ReadStream(
			ctx,
			auditStream,
			auditGroup,
			auditConsumer,
		)

		if err != nil {
			// 🔥 KLUCZ: SAMO-NAPRAWA
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

		for _, entry := range entries {
			w.logger.InfoMap("Worker: Received entry", map[string]any{
				"id": entry.ID,
			})

			rawPayload, ok := entry.Values["payload"].(string)
			if !ok {
				w.logger.WarnMap("Worker: payload missing or invalid", map[string]any{
					"entry_id": entry.ID,
				})
				continue
			}

			w.logger.Debug("Worker: Raw payload: " + rawPayload)
			w.logger.DebugMap("Worker: Parsed payload preview", map[string]any{
				"payload_preview": rawPayload,
			})

			var msg AuditMessage
			if err := json.Unmarshal([]byte(rawPayload), &msg); err != nil {
				w.logger.ErrorObj("Worker: JSON unmarshal failed", err)
				continue
			}

			w.logger.DebugMap("Worker: AuditMessage parsed", map[string]any{
				"user_id": msg.UserID,
				"service": msg.Service,
				"action":  msg.Action,
			})

			if err := w.svc.SaveLog(ctx, msg); err != nil {
				w.logger.ErrorObj("Worker: SaveLog failed", err)
				continue
			}

			if err := w.redis.AckStream(
				ctx,
				auditStream,
				auditGroup,
				entry.ID,
			); err != nil {
				w.logger.ErrorObj("Worker: ACK failed", err)
				continue
			}

			w.logger.Info("Worker: Log processed and ACKed: " + entry.ID)
		}
	}
}

// =======================================================
// 🔧 INFRA SELF-HEALING
// =======================================================

func (w *AuditWorker) ensureRedisInfrastructure(ctx context.Context) error {
	// 1️⃣ Wymuś istnienie streama (XADD noop)
	if err := w.redis.SendAuditLog(ctx, map[string]any{
		"_bootstrap": true,
		"_ts":        time.Now().Unix(),
	}); err != nil {
		return err
	}

	// 2️⃣ Zapewnij istnienie consumer group
	if err := w.redis.EnsureGroup(ctx, auditStream, auditGroup); err != nil {
		return err
	}

	return nil
}
