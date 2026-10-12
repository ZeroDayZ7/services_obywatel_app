package publisher

import (
	"context"

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
	event := make(Event, len(payload)+3)
	for key, value := range payload {
		event[key] = value
	}
	event["source_service"] = SourceService
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
		event["target_user_ids"] = users
	} else {
		event["target_user_ids"] = []string{}
	}
	return event
}
