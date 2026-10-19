package publisher

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewEventAddsWebSocketEnvelopeFields(t *testing.T) {
	ownerID := uuid.New()
	targetID := uuid.New()
	event := NewEvent([]uuid.UUID{ownerID, targetID}, map[string]any{
		"type":       "contact_updated",
		"contact_id": uuid.NewString(),
	})

	if event["sourceService"] != SourceService {
		t.Fatalf("expected sourceService %q, got %v", SourceService, event["sourceService"])
	}
	if event["type"] != "contact_updated" {
		t.Fatalf("expected event type contact_updated, got %v", event["type"])
	}

	targetUsers, ok := event["targetUserIds"].([]string)
	if !ok {
		t.Fatalf("expected targetUserIds to be []string, got %#v", event["targetUserIds"])
	}
	if len(targetUsers) != 2 {
		t.Fatalf("expected 2 users in fan-out, got %d", len(targetUsers))
	}
	if targetUsers[0] != ownerID.String() || targetUsers[1] != targetID.String() {
		t.Fatalf("expected exact fan-out users, got %#v", targetUsers)
	}
	if _, exists := event["target_user_ids"]; exists {
		t.Fatal("legacy target_user_ids key must not exist")
	}
	if _, exists := event["source_service"]; exists {
		t.Fatal("legacy source_service key must not exist")
	}
	if _, exists := event["eventType"]; exists {
		t.Fatal("duplicate eventType key must not exist")
	}
}
