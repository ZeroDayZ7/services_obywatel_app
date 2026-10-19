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
	event := make(Event, len(payload)+8)
	maps.Copy(event, payload)

	if _, exists := event["sourceService"]; !exists {
		event["sourceService"] = SourceService
	}

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
		} else {
			event["targetUserIds"] = []string{}
		}
	}
	return event
}
