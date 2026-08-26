// cmdr: cmdr: context\context.go

package context

import (
	"context"

	"github.com/google/uuid"
)

type RequestContext struct {
	RequestID   string
	UserID      *uuid.UUID
	SessionID   *uuid.UUID
	DeviceID    string
	IP          string
	Role        string
	Permissions []string
	RiskScore   int
	Challenge   string

	InstitutionID *uuid.UUID
	DepartmentID  *uuid.UUID
	Username      string
}

//#region FromContext
func FromContext(ctx context.Context) (*RequestContext, bool) {
	reqCtx, ok := ctx.Value(RequestContextKey).(*RequestContext)
	return reqCtx, ok
}

//#region GetIP
func GetIP(ctx context.Context) string {
	if reqCtx, ok := FromContext(ctx); ok && reqCtx != nil {
		return reqCtx.IP
	}
	return ""
}

//#region GetUserID
func GetUserID(ctx context.Context) uuid.UUID {
	if reqCtx, ok := FromContext(ctx); ok && reqCtx != nil && reqCtx.UserID != nil {
		return *reqCtx.UserID
	}
	return uuid.Nil
}
