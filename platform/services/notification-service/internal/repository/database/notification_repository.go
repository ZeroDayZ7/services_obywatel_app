package mysql

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/notification-service/internal/model"
	"gorm.io/gorm"
)

type NotificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// Używamy db.WithContext(ctx), aby GORM wiedział o timeoutach i przerwanych połączeniach

func (r *NotificationRepository) Create(ctx context.Context, n *model.Notification) error {
	return r.db.WithContext(ctx).Create(n).Error
}

func (r *NotificationRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]model.Notification, error) {
	var notifications []model.Notification
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Order("created_at DESC").
		Find(&notifications).Error
	return notifications, err
}

func (r *NotificationRepository) MarkAsRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("is_read", true).Error
}

func (r *NotificationRepository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Update("is_read", true).Error
}

func (r *NotificationRepository) MoveToTrash(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("deleted_at", gorm.Expr("NOW()")).Error
}

func (r *NotificationRepository) RestoreFromTrash(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("deleted_at", nil).Error
}

func (r *NotificationRepository) HardDeleteTrash(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND deleted_at IS NOT NULL", userID).
		Delete(&model.Notification{}).Error
}

func (r *NotificationRepository) DeletePermanently(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&model.Notification{}).Error
}

// ProcessSyncBatch applies a batch of sync events in a single DB transaction.
func (r *NotificationRepository) ProcessSyncBatch(ctx context.Context, userID uuid.UUID, req model.SyncBatchRequest) (processed []string, failed []string, err error) {
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	for _, ev := range req.Events {
		switch ev.EventType {
		case "notification.mark_read":
			if ev.Payload != nil {
				var p map[string]string
				if err := json.Unmarshal(ev.Payload, &p); err == nil {
					idStr := p["id"]
					if id, parseErr := uuid.Parse(idStr); parseErr == nil {
						if uErr := tx.WithContext(ctx).
							Model(&model.Notification{}).
							Where("id = ? AND user_id = ?", id, userID).
							Update("is_read", true).Error; uErr == nil {
							processed = append(processed, ev.ID)
							continue
						}
					}
				}
			}
			failed = append(failed, ev.ID)
		case "notification.mark_all_read":
			if uErr := tx.WithContext(ctx).
				Model(&model.Notification{}).
				Where("user_id = ? AND is_read = ?", userID, false).
				Update("is_read", true).Error; uErr == nil {
				processed = append(processed, ev.ID)
				continue
			}
			failed = append(failed, ev.ID)
		case "notification.move_to_trash":
			if ev.Payload != nil {
				var p map[string]string
				if err := json.Unmarshal(ev.Payload, &p); err == nil {
					idStr := p["id"]
					if id, parseErr := uuid.Parse(idStr); parseErr == nil {
						if uErr := tx.WithContext(ctx).
							Model(&model.Notification{}).
							Where("id = ? AND user_id = ?", id, userID).
							Update("deleted_at", gorm.Expr("NOW()")).Error; uErr == nil {
							processed = append(processed, ev.ID)
							continue
						}
					}
				}
			}
			failed = append(failed, ev.ID)
		case "notification.restore":
			if ev.Payload != nil {
				var p map[string]string
				if err := json.Unmarshal(ev.Payload, &p); err == nil {
					idStr := p["id"]
					if id, parseErr := uuid.Parse(idStr); parseErr == nil {
						if uErr := tx.WithContext(ctx).
							Model(&model.Notification{}).
							Where("id = ? AND user_id = ?", id, userID).
							Update("deleted_at", nil).Error; uErr == nil {
							processed = append(processed, ev.ID)
							continue
						}
					}
				}
			}
			failed = append(failed, ev.ID)
		case "notification.delete":
			if ev.Payload != nil {
				var p map[string]string
				if err := json.Unmarshal(ev.Payload, &p); err == nil {
					idStr := p["id"]
					if id, parseErr := uuid.Parse(idStr); parseErr == nil {
						if uErr := tx.WithContext(ctx).
							Where("id = ? AND user_id = ?", id, userID).
							Delete(&model.Notification{}).Error; uErr == nil {
							processed = append(processed, ev.ID)
							continue
						}
					}
				}
			}
			failed = append(failed, ev.ID)
		case "notification.clear_trash":
			if uErr := tx.WithContext(ctx).
				Where("user_id = ? AND deleted_at IS NOT NULL", userID).
				Delete(&model.Notification{}).Error; uErr == nil {
				processed = append(processed, ev.ID)
				continue
			}
			failed = append(failed, ev.ID)
		default:
			// Unknown event -> mark failed
			failed = append(failed, ev.ID)
		}
	}

	if len(failed) > 0 {
		tx.Rollback()
		return processed, failed, nil
	}

	if err := tx.Commit().Error; err != nil {
		return processed, failed, err
	}

	return processed, failed, nil
}
