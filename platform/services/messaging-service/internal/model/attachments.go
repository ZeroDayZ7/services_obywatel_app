package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type AttachmentStatus string

const (
	AttachmentStatusPending  AttachmentStatus = "pending"
	AttachmentStatusUploaded AttachmentStatus = "uploaded"
	AttachmentStatusAttached AttachmentStatus = "attached"
	AttachmentStatusDeleted  AttachmentStatus = "deleted"
)

type Attachment struct {
	ID             uuid.UUID        `gorm:"type:uuid;primaryKey;default:uuidv7()" json:"id"`
	UserID         uuid.UUID        `gorm:"type:uuid;not null;index" json:"userId"`
	ConversationID *uuid.UUID       `gorm:"type:uuid;index" json:"conversationId,omitempty"`
	MessageID      *uuid.UUID       `gorm:"type:uuid;index" json:"messageId,omitempty"`
	FileKey        string           `gorm:"type:varchar(512);not null" json:"fileKey"`
	OriginalName   string           `gorm:"type:varchar(255);not null" json:"originalName"`
	ContentType    string           `gorm:"type:varchar(128);not null" json:"contentType"`
	Size           int64            `gorm:"not null;default:0" json:"size"`
	Sha256         string           `gorm:"type:varchar(128)" json:"sha256,omitempty"`
	Status         AttachmentStatus `gorm:"type:varchar(24);not null;default:'pending';index" json:"status"`
	Metadata       datatypes.JSON   `gorm:"type:jsonb" json:"metadata,omitempty"`

	BaseModel
}

type AttachmentUploadRequest struct {
	ConversationID *uuid.UUID `json:"conversationId,omitempty"`
	FileName       string     `json:"fileName"`
	ContentType    string     `json:"contentType"`
	Size           int64      `json:"size"`
}

type AttachmentFinalizeRequest struct {
	AttachmentID uuid.UUID `json:"attachmentId"`
	Size         int64     `json:"size"`
	Sha256       string    `json:"sha256,omitempty"`
}

type AttachmentUploadResponse struct {
	Attachment *Attachment `json:"attachment"`
	UploadURL  string      `json:"uploadUrl"`
	ExpiresAt  time.Time   `json:"expiresAt"`
}
