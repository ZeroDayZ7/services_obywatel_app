package publisher

import (
	"context"
	"maps"

	"github.com/google/uuid"
)

const (
	WebsocketChannel = "ws:events"
	SourceService    = "messaging-service"
)

type Event map[string]any

type EventPublisher interface {
	Publish(ctx context.Context, event Event) error
}

func NewEvent(targetUserIDs []uuid.UUID, payload map[string]any) Event {
	event := make(Event, len(payload)+10)
	maps.Copy(event, payload)

	if eventType, exists := payload["type"]; exists {
		event["type"] = eventType
		event["eventType"] = eventType
	} else if eventType, exists := payload["eventType"]; exists {
		event["type"] = eventType
		event["eventType"] = eventType
	}
	if _, exists := payload["sourceService"]; !exists {
		event["sourceService"] = SourceService
	}
	event["source_service"] = SourceService

	if _, exists := event["targetUserIds"]; !exists {
		if len(targetUserIDs) > 0 {
			users := make([]string, 0, len(targetUserIDs))
			seen := make(map[string]struct{}, len(targetUserIDs))
			for _, id := range targetUserIDs {
				if id == uuid.Nil {
					continue
				}
				userID := id.String()
				if _, exists := seen[userID]; exists {
					continue
				}
				seen[userID] = struct{}{}
				users = append(users, userID)
			}
			event["targetUserIds"] = users
			event["target_user_ids"] = users
		} else {
			event["targetUserIds"] = []string{}
			event["target_user_ids"] = []string{}
		}
	}
	if _, exists := event["eventId"]; !exists {
		if value, ok := event["event_id"]; ok {
			event["eventId"] = value
		}
	}
	if _, exists := event["conversationId"]; !exists {
		if value, ok := event["conversation_id"]; ok {
			event["conversationId"] = value
		}
	}
	if _, exists := event["senderUserId"]; !exists {
		if value, ok := event["sender_user_id"]; ok {
			event["senderUserId"] = value
		}
	}
	if _, exists := event["messageId"]; !exists {
		if value, ok := event["message_id"]; ok {
			event["messageId"] = value
		}
	}
	return event
}
