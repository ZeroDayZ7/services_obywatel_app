package events

import "time"

// EventType – jawne, wersjonowalne typy eventów
type EventType string

const (
	// Auth / Security
	DeviceRegistered EventType = "DEVICE_REGISTERED"
	LoginSuccess     EventType = "LOGIN_SUCCESS"
	LoginFailed      EventType = "LOGIN_FAILED"
	Logout           EventType = "LOGOUT"

	// Account
	PasswordChanged EventType = "PASSWORD_CHANGED"
	EmailChanged    EventType = "EMAIL_CHANGED"
)

// Event – neutralny event systemowy
type Event struct {
	ID        string         `json:"id"`
	Type      EventType      `json:"type"`
	UserID    string         `json:"user_id"`
	Source    string         `json:"source"` // auth-service, profile-service…
	IP        string         `json:"ip,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Flags     EventFlags     `json:"flags"`
	Version   int            `json:"version"`
}
