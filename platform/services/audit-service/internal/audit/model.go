package audit

import "time"

type AuditMessage struct {
	UserID    int64                  `json:"user_id"`
	Service   string                 `json:"service"`    // np. "Audit-service", "payment-service"
	Action    string                 `json:"action"`     // np. "login_attempt", "payout_created"
	IPAddress string                 `json:"ip_address"` // Jawne, do filtrowania w DB
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"` // Szczegóły - to zostanie zaszyfrowane
}

// AuditLogResponse to struktura zwracana do panelu Admina.
// Używamy tagów JSON, aby Flutter wiedział jak to mapować.
type AuditLogResponse struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	Action        string    `json:"action"`
	EncryptedData []byte    `json:"encrypted_data"` // Base64 we Flutterze
	EncryptedKey  []byte    `json:"encrypted_key"`  // Base64 we Flutterze
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}
