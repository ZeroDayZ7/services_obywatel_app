package model

import (
	"time"

	"github.com/google/uuid"
)

type AuditLogResponse struct {
	ID          uuid.UUID              `json:"id"`
	UserID      uuid.UUID              `json:"user_id"`
	ServiceName string                 `json:"service_name"`
	Action      string                 `json:"action"`
	IPAddress   string                 `json:"ip_address,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Status      string                 `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
}

type CreateAuditLogRequest struct {
	UserID      uuid.UUID              `json:"user_id"`
	ServiceName string                 `json:"service_name"`
	Action      string                 `json:"action"`
	IPAddress   string                 `json:"ip_address,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Status      string                 `json:"status"`
}
