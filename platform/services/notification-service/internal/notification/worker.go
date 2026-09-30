package notification

import (
	"context"
	"encoding/json"
	"time"

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
	redis   *redis.Client
	svc     *service.NotificationService
	logger  *shared.Logger
	enabled bool
}

func NewNotificationWorker(r *redis.Client, s *service.NotificationService, l *shared.Logger) *NotificationWorker {
	return &NotificationWorker{
		redis:   r,
		svc:     s,
		logger:  l,
		enabled: r != nil,
	}
}

func (w *NotificationWorker) SetEnabled(enabled bool) {
	w.enabled = enabled
}

func (w *NotificationWorker) isEnabled() bool {
	return w.enabled && w.redis != nil
}

func (w *NotificationWorker) shouldSkipRedis() bool {
	return !w.isEnabled()
}

func (w *NotificationWorker) Start() {
	if w.shouldSkipRedis() {
		if w.logger != nil {
			w.logger.Warn("NotificationWorker: disabled or redis unavailable, skipping stream consumer")
		}
		return
	}

	ctx := context.Background()

	if err := w.ensureRedisInfrastructure(ctx); err != nil {
		w.logger.ErrorObj("NotificationWorker: failed to bootstrap redis infra", err)
		return
	}

	w.logger.Info("NotificationWorker: Listening for events...")

	for {
		entries, err := w.redis.ReadStream(ctx, notificationStream, notificationGroup, notificationConsumer)
		if err != nil {
			// ... (obsługa błędów bez zmian)
			time.Sleep(5 * time.Second)
			continue
		}

		for _, entry := range entries {
			rawPayload, ok := entry.Values["payload"].(string)
			if !ok {
				continue
			}

			var evt model.NotificationEvent
			if err := json.Unmarshal([]byte(rawPayload), &evt); err != nil {
				w.logger.ErrorObj("NotificationWorker: JSON unmarshal failed", err)
				continue
			}

			// LOGIKA: Nie ustawiamy ID, CreatedAt ani IsRead ręcznie.
			// Model zrobi to sam w BeforeCreate podczas s.svc.Send(ctx, notification)
			notification := &model.Notification{
				UserID:   evt.UserID,
				Title:    evt.Title,
				Content:  evt.Content,
				Priority: evt.Priority,
				Category: evt.Category,
			}

			if err := w.svc.Send(ctx, notification); err != nil {
				w.logger.ErrorObj("NotificationWorker: failed to save notification", err)
				continue
			}

			_ = w.redis.AckStream(ctx, notificationStream, notificationGroup, entry.ID)
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
