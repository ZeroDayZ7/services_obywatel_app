// cmdr: cmdr: context\context.go

package context

import (
	"context"

	"github.com/google/uuid"
)

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

func FromContext(ctx context.Context) (*RequestContext, bool) {
	reqCtx, ok := ctx.Value(RequestContextKey).(*RequestContext)
	return reqCtx, ok
}

func GetIP(ctx context.Context) string {
	if reqCtx, ok := FromContext(ctx); ok && reqCtx != nil {
		return reqCtx.IP
	}
	return ""
}
