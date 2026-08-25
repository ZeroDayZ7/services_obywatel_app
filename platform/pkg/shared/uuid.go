// platform/pkg/shared/uuid.go
package shared

import (
	"github.com/google/uuid"
)

// NewUUIDv7
//#region NewUUIDv7
func NewUUIDv7() uuid.UUID {
	return uuid.Must(uuid.NewV7())
}

// GenerateSessionID
//#region GenerateSessionID
func GenerateSessionID() string {
	return NewUUIDv7().String()
}
