package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/shared"
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
	streamName := "audit_stream"
	groupName := "audit_service_group"
	consumerName := "worker_1"

	w.logger.Info("Audit Worker started and listening on stream: " + streamName)

	for {
		entries, err := w.redis.ReadStream(ctx, streamName, groupName, consumerName)
		if err != nil {
			w.logger.ErrorObj("Failed to read from redis stream", err)
			time.Sleep(2 * time.Second)
			continue
		}

		for _, entry := range entries {
			var msg AuditMessage

			if rawPayload, ok := entry.Values["payload"].(string); ok {
				err := json.Unmarshal([]byte(rawPayload), &msg)
				if err != nil {
					w.logger.ErrorObj("Failed to unmarshal audit message JSON", err)
					continue
				}
			} else {
				w.logger.Warn("Received message without 'payload' field")
				continue
			}

			err = w.svc.SaveLog(ctx, msg)
			if err != nil {
				w.logger.ErrorObj("Failed to process audit log", err)
				continue
			}

			w.redis.AckStream(ctx, streamName, groupName, entry.ID)
		}
	}
}
