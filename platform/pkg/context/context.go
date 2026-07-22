// cmdr: cmdr: context\context.go

package context

import "github.com/google/uuid"

type RequestContext struct {
	RequestID   string
	UserID      *uuid.UUID
	SessionID   string
	DeviceID    string
	IP          string
	Role        string
	Permissions []string
	RiskScore   int
	Challenge   string
}
