package audit

import "time"

type AuditMessage struct {
	UserID    string         `json:"user_id"`
	Service   string         `json:"service"`
	Action    string         `json:"action"`
	IPAddress string         `json:"ip"`
	Timestamp *time.Time     `json:"timestamp,omitempty"`
	Metadata  map[string]any `json:"metadata"`
}

type AuditLogResponse struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	Action        string    `json:"action"`
	EncryptedData []byte    `json:"encrypted_data"`
	EncryptedKey  []byte    `json:"encrypted_key"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}
