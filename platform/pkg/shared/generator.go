package shared

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// #region GenerateAgreementNumber

// GenerateAgreementNumber tworzy unikalny numer umowy na podstawie daty i znacznika czasu.
func GenerateAgreementNumber(t time.Time) string {
	return fmt.Sprintf("AGR/%s/%d", t.Format("20060102"), t.Unix()%100000)
}

// NewUUIDv7
// #region NewUUIDv7
func NewUUIDv7() uuid.UUID {
	return uuid.Must(uuid.NewV7())
}

// GenerateSessionID
// #region GenerateSessionID
func GenerateSessionID() uuid.UUID {
	return NewUUIDv7()
}
