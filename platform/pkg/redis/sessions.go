// cmdr: redis/sessions.go

package redis

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/zerodayz7/platform/pkg/constants"
)

type UserSession struct {
	UserID      string `json:"user_id"`
	Username    string `json:"username,omitempty"`
	Email       string `json:"email,omitempty"`
	Fingerprint string `json:"fingerprint"`
	Role        string `json:"role,omitempty"`
	PublicKey   string `json:"public_key,omitempty"`
	Challenge   string `json:"challenge,omitempty"`
	IP          string `json:"ip,omitempty"`

	// Metadane pracownika
	EmployeeNumber string   `json:"employee_number,omitempty"`
	InstitutionID  string   `json:"institution_id,omitempty"`
	DepartmentID   string   `json:"department_id,omitempty"`
	Permissions    []string `json:"permissions,omitempty"`
}

// --- Metody dla Sesji Głównej ---

// #region SetSession
func (c *Cache) SetSession(ctx context.Context, sid uuid.UUID, session *UserSession, ttl time.Duration) error {
	return SetJSON(c, ctx, constants.SessionPrefix+sid.String(), session, ttl)
}

// #region GetSession
func (c *Cache) GetSession(ctx context.Context, sid uuid.UUID) (*UserSession, error) {
	return GetJSON[UserSession](c, ctx, constants.SessionPrefix+sid.String())
}

// #region DeleteSession
func (c *Cache) DeleteSession(ctx context.Context, sid uuid.UUID) error {
	return c.Del(ctx, constants.SessionPrefix+sid.String())
}

// #region UpdateSession
func (c *Cache) UpdateSession(ctx context.Context, sid uuid.UUID, updateFn func(*UserSession)) error {
	session, err := c.GetSession(ctx, sid)
	if err != nil {
		return err
	}

	updateFn(session)

	ttl, _ := c.client.TTL(ctx, constants.SessionPrefix+sid.String()).Result()
	if ttl <= 0 {
		ttl = c.ttl
	}

	return c.SetSession(ctx, sid, session, ttl)
}

// --- Metody dla Challenge (Ed25519) ---

// #region SetChallenge
func (c *Cache) SetChallenge(ctx context.Context, sid uuid.UUID, challenge string, ttl time.Duration) error {
	return c.Set(ctx, constants.ChallengePrefix+sid.String(), challenge, ttl)
}

// #region GetChallenge
func (c *Cache) GetChallenge(ctx context.Context, sid uuid.UUID) (string, error) {
	return c.Get(ctx, constants.ChallengePrefix+sid.String())
}

// #region DeleteChallenge
func (c *Cache) DeleteChallenge(ctx context.Context, sid uuid.UUID) error {
	return c.Del(ctx, constants.ChallengePrefix+sid.String())
}

// --- Metody dla Sesji Tymczasowej (Setup/2FA) ---

// #region SetSetupSession
func (c *Cache) SetSetupSession(ctx context.Context, sid uuid.UUID, session *UserSession, ttl time.Duration) error {
	return SetJSON(c, ctx, constants.SetupSessionPrefix+sid.String(), session, ttl)
}

// #region GetSetupSession
func (c *Cache) GetSetupSession(ctx context.Context, sid uuid.UUID) (*UserSession, error) {
	return GetJSON[UserSession](c, ctx, constants.SetupSessionPrefix+sid.String())
}

// #region DeleteSetupSession
func (c *Cache) DeleteSetupSession(ctx context.Context, sid uuid.UUID) error {
	return c.Del(ctx, constants.SetupSessionPrefix+sid.String())
}
