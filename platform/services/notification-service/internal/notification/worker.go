package notification

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/notification-service/internal/model"
	"github.com/zerodayz7/platform/services/notification-service/internal/service"
)

const (
	notificationStream   = "notification_stream"
	notificationGroup    = "notification_service_group"
	notificationConsumer = "worker_1"
)

type NotificationWorker struct {
	redis  *redis.Client
	svc    *service.NotificationService
	logger *shared.Logger
}

func NewNotificationWorker(r *redis.Client, s *service.NotificationService, l *shared.Logger) *NotificationWorker {
	return &NotificationWorker{
		redis:  r,
		svc:    s,
		logger: l,
	}
}

func (w *NotificationWorker) Start() {
	ctx := context.Background()

	// 🔧 BOOTSTRAP: zapewnij istnienie grupy i streama
	if err := w.ensureRedisInfrastructure(ctx); err != nil {
		w.logger.ErrorObj("NotificationWorker: failed to bootstrap redis infra", err)
		return
	}

	w.logger.Info("NotificationWorker: Listening for events...")

	for {
		entries, err := w.redis.ReadStream(
			ctx,
			notificationStream,
			notificationGroup,
			notificationConsumer,
		)

		if err != nil {
			if strings.Contains(err.Error(), "NOGROUP") {
				w.logger.Warn("NotificationWorker: consumer group missing, recreating...")
				if err := w.ensureRedisInfrastructure(ctx); err != nil {
					w.logger.ErrorObj("NotificationWorker: failed to recreate redis infra", err)
					time.Sleep(5 * time.Second)
				}
				continue
			}

			w.logger.ErrorObj("NotificationWorker: Redis error", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if len(entries) == 0 {
			continue
		}

		for _, entry := range entries {
			rawPayload, ok := entry.Values["payload"].(string)
			if !ok {
				w.logger.WarnMap("NotificationWorker: payload missing or invalid", map[string]any{
					"entry_id": entry.ID,
				})
				continue
			}

			var evt model.NotificationEvent
			if err := json.Unmarshal([]byte(rawPayload), &evt); err != nil {
				w.logger.ErrorObj("NotificationWorker: JSON unmarshal failed", err)
				continue
			}

			notification := &model.Notification{
				ID:        uuid.New(),
				UserID:    evt.UserID,
				Title:     evt.Title,
				Content:   evt.Content,
				Priority:  evt.Priority,
				Category:  evt.Category,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				IsRead:    false,
			}

			if err := w.svc.Send(notification); err != nil {
				w.logger.ErrorObj("NotificationWorker: failed to save notification", err)
				continue
			}

			// ACK
			if err := w.redis.AckStream(ctx, notificationStream, notificationGroup, entry.ID); err != nil {
				w.logger.ErrorObj("NotificationWorker: ACK failed", err)
				continue
			}

			w.logger.Info("NotificationWorker: notification processed and ACKed: " + entry.ID)
		}
	}
}

func (w *NotificationWorker) ensureRedisInfrastructure(ctx context.Context) error {
	// 1️⃣ wymuś istnienie streama
	if err := w.redis.SendNotification(ctx, map[string]any{
		"_bootstrap": true,
		"_ts":        time.Now().Unix(),
	}); err != nil {
		return err
	}

	// 2️⃣ consumer group
	if err := w.redis.EnsureGroup(ctx, notificationStream, notificationGroup); err != nil {
		return err
	}

	return nil
}
