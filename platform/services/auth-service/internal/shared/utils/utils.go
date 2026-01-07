package utils

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/shared"
)

type NotificationOptions struct {
	Title    string
	Priority string
	Category string
}

func SendEvent(ctx context.Context, cache *redis.Cache, userID uuid.UUID, action string, metadata map[string]any, clientIP string, opts ...NotificationOptions) {
	log := shared.GetLogger()

	// Domyślne wartości
	title := "System Alert"
	priority := "info"
	category := "general"

	// Jeśli przekazano opcje, nadpisujemy domyślne
	if len(opts) > 0 {
		if opts[0].Title != "" {
			title = opts[0].Title
		}
		if opts[0].Priority != "" {
			priority = opts[0].Priority
		}
		if opts[0].Category != "" {
			category = opts[0].Category
		}
	}

	auditEvent := map[string]any{
		"user_id":  userID,
		"action":   action,
		"metadata": metadata,
		"service":  "auth-service",
		"ip":       clientIP,
	}

	notificationEvent := map[string]any{
		"user_id":  userID,
		"title":    title,
		"content":  fmt.Sprintf("Event: %s", action),
		"priority": priority,
		"category": category,
		"metadata": metadata,
	}

	go func() {
		if err := cache.SendAuditLog(ctx, auditEvent); err != nil {
			log.ErrorObj("Failed to send audit log", err)
		}
	}()

	go func() {
		if err := cache.SendNotification(ctx, notificationEvent); err != nil {
			log.ErrorObj("Failed to send notification", err)
		}
	}()
}
