package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OutboxStatus string

const (
	OutboxStatusPending    OutboxStatus = "PENDING"
	OutboxStatusProcessing OutboxStatus = "PROCESSING"
	OutboxStatusSent       OutboxStatus = "SENT"
	OutboxStatusFailed     OutboxStatus = "FAILED"
)

type OutboxMessage struct {
	ID            uuid.UUID    `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	AggregateType string       `gorm:"size:64;not null;index"`   // np. "CITIZEN", "USER_AGREEMENT"
	AggregateID   uuid.UUID    `gorm:"type:uuid;not null;index"` // np. Citizen.UserID
	EventType     string       `gorm:"size:128;not null;index"`  // np. "CitizenRegistered", "AgreementSigned"
	Payload       []byte       `gorm:"type:jsonb;not null"`      // Lub []byte (blob) w zależności od DB
	Status        OutboxStatus `gorm:"type:varchar(20);not null;default:'PENDING';index"`
	RetryCount    int8         `gorm:"not null;default:0"`
	LastError     *string      `gorm:"type:text"`
	ProcessedAt   *time.Time
	CreatedAt     time.Time      `gorm:"autoCreateTime;index"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}
