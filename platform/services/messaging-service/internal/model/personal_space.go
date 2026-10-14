package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type PersonalSpace struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	OwnerID     uuid.UUID `gorm:"type:uuid;not null;index;unique" json:"ownerId"`
	Title       string    `gorm:"type:varchar(128);not null;default:'Notatki dla siebie'" json:"title"`
	SyncVersion uint64    `gorm:"not null;default:0" json:"syncVersion"`
	BaseModel
}

type PersonalSpaceItemType string

const (
	PersonalSpaceItemTypeText PersonalSpaceItemType = "text"
	PersonalSpaceItemTypeFile PersonalSpaceItemType = "file"
)

type PersonalSpaceItemStatus string

const (
	PersonalSpaceItemStatusActive  PersonalSpaceItemStatus = "active"
	PersonalSpaceItemStatusDeleted PersonalSpaceItemStatus = "deleted"
)

type PersonalSpaceItem struct {
	ID               uuid.UUID               `gorm:"type:uuid;primaryKey" json:"id"`
	SpaceID          uuid.UUID               `gorm:"type:uuid;index;not null" json:"spaceId"`
	OwnerID          uuid.UUID               `gorm:"type:uuid;index;not null" json:"ownerId"`
	DeviceID         string                  `gorm:"type:varchar(64);not null" json:"deviceId"`
	Title            string                  `gorm:"type:varchar(256)" json:"title,omitempty"`
	ContentType      PersonalSpaceItemType   `gorm:"type:varchar(32);not null;default:'text'" json:"contentType"`
	EncryptedPayload []byte                  `gorm:"type:bytea;not null" json:"encryptedPayload"`
	Status           PersonalSpaceItemStatus `gorm:"type:varchar(20);not null;default:'active';index" json:"status"`
	SyncVersion      uint64                  `gorm:"not null;default:1" json:"syncVersion"`
	Metadata         datatypes.JSON          `gorm:"type:jsonb;default:'{}'" json:"metadata,omitempty"`
	Source           string                  `gorm:"type:varchar(32);not null;default:'device'" json:"source"`
	CreatedAt        time.Time               `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt        time.Time               `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt        gorm.DeletedAt          `gorm:"index" json:"deletedAt,omitempty"`
}

type CreatePersonalSpaceItemRequest struct {
	SpaceID          *uuid.UUID            `json:"spaceId,omitempty"`
	Title            string                `json:"title,omitempty"`
	ContentType      PersonalSpaceItemType `json:"contentType"`
	EncryptedPayload []byte                `json:"encryptedPayload"`
	DeviceID         string                `json:"deviceId,omitempty"`
	Metadata         map[string]any        `json:"metadata,omitempty"`
}

type UpdatePersonalSpaceItemRequest struct {
	ItemID           uuid.UUID             `json:"itemId"`
	Title            string                `json:"title,omitempty"`
	ContentType      PersonalSpaceItemType `json:"contentType,omitempty"`
	EncryptedPayload []byte                `json:"encryptedPayload,omitempty"`
	DeviceID         string                `json:"deviceId,omitempty"`
	Metadata         map[string]any        `json:"metadata,omitempty"`
	SyncVersion      uint64                `json:"syncVersion,omitempty"`
}

type PersonalSpaceListResponse struct {
	SpaceID     uuid.UUID           `json:"spaceId"`
	OwnerID     uuid.UUID           `json:"ownerId"`
	Items       []PersonalSpaceItem `json:"items"`
	SyncVersion uint64              `json:"syncVersion"`
	UpdatedAt   time.Time           `json:"updatedAt"`
}
