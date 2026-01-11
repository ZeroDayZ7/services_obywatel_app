package http

import "time"

type SessionResponse struct {
	ID        uint       `json:"id"`
	Device    string     `json:"device_name"`
	Platform  string     `json:"platform"`
	IsCurrent bool       `json:"is_current"`
	CreatedAt time.Time  `json:"created_at"`
	LastUsed  *time.Time `json:"last_used,omitempty"`
}

type TerminateSessionResponse struct {
	Status string `json:"status"`
}
