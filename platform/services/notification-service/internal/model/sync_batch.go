package model

import (
	"encoding/json"
)

type SyncEvent struct {
    ID        string          `json:"id"`
    EventType string          `json:"event_type"`
    Payload   json.RawMessage `json:"payload"`
    CreatedAt string          `json:"created_at"`
}

type SyncBatchRequest struct {
    Events []SyncEvent `json:"events"`
}

type SyncBatchResponse struct {
    ProcessedEventIDs []string `json:"processed_event_ids"`
    FailedEventIDs    []string `json:"failed_event_ids,omitempty"`
}
