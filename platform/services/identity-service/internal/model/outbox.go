package model

import (
	"time"

	"github.com/google/uuid"
)

type OutboxStatus string

const (
	OutboxStatusPending    OutboxStatus = "PENDING"
	OutboxStatusProcessing OutboxStatus = "PROCESSING"
	OutboxStatusSent       OutboxStatus = "SENT"
	OutboxStatusFailed     OutboxStatus = "FAILED"
)

type OutboxMessage struct {
	ID            uuid.UUID    `json:"id"`
	AggregateType string       `json:"aggregate_type"` // np. "CITIZEN", "USER_AGREEMENT"
	AggregateID   uuid.UUID    `json:"aggregate_id"`   // np. Citizen.UserID
	EventType     string       `json:"event_type"`     // np. "CitizenRegistered", "AgreementSigned"
	Payload       []byte       `json:"payload"`
	Status        OutboxStatus `json:"status"`
	RetryCount    int8         `json:"retry_count"`
	LastError     *string      `json:"last_error,omitempty"`
	ProcessedAt   *time.Time   `json:"processed_at,omitempty"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
	DeletedAt     *time.Time   `json:"deleted_at,omitempty"`
}
