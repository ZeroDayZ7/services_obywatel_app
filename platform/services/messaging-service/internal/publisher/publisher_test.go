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

	if event["source_service"] != SourceService {
		t.Fatalf("expected source_service %q, got %v", SourceService, event["source_service"])
	}
	if event["type"] != "contact_updated" {
		t.Fatalf("expected event type contact_updated, got %v", event["type"])
	}

	targetUsers, ok := event["target_user_ids"].([]string)
	if !ok {
		t.Fatalf("expected target_user_ids to be []string, got %#v", event["target_user_ids"])
	}
	if len(targetUsers) != 2 {
		t.Fatalf("expected 2 users in fan-out, got %d", len(targetUsers))
	}
	if targetUsers[0] != ownerID.String() || targetUsers[1] != targetID.String() {
		t.Fatalf("expected exact fan-out users, got %#v", targetUsers)
	}
}
