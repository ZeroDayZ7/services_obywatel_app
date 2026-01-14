package context

import "github.com/google/uuid"

type RequestContext struct {
	RequestID string
	UserID    *uuid.UUID
	SessionID string
	DeviceID  string
	IP        string
	Roles     []string
	RiskScore int
	Challenge string
}
