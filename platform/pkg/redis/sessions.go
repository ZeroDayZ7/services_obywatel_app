// cmdr: redis/sessions.go

package redis

import (
	"context"
	"time"

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

func (c *Cache) SetSession(ctx context.Context, sid string, sess UserSession, ttl time.Duration) error {
	return SetJSON(c, ctx, constants.SessionPrefix+sid, sess, ttl)
}

func (c *Cache) GetSession(ctx context.Context, sid string) (*UserSession, error) {
	return GetJSON[UserSession](c, ctx, constants.SessionPrefix+sid)
}

func (c *Cache) DeleteSession(ctx context.Context, sid string) error {
	return c.Del(ctx, constants.SessionPrefix+sid)
}

func (c *Cache) UpdateSession(ctx context.Context, sid string, updateFn func(*UserSession)) error {
	session, err := c.GetSession(ctx, sid)
	if err != nil {
		return err
	}

	updateFn(session)

	ttl, _ := c.client.TTL(ctx, constants.SessionPrefix+sid).Result()
	if ttl <= 0 {
		ttl = c.ttl
	}

	return c.SetSession(ctx, sid, *session, ttl)
}

// --- Metody dla Challenge (Ed25519) ---

func (c *Cache) SetChallenge(ctx context.Context, sid string, challenge string, ttl time.Duration) error {
	return c.Set(ctx, constants.ChallengePrefix+sid, challenge, ttl)
}

func (c *Cache) GetChallenge(ctx context.Context, sid string) (string, error) {
	return c.Get(ctx, constants.ChallengePrefix+sid)
}

func (c *Cache) DeleteChallenge(ctx context.Context, sid string) error {
	return c.Del(ctx, constants.ChallengePrefix+sid)
}

// --- Metody dla Sesji Tymczasowej (Setup/2FA) ---

func (c *Cache) SetSetupSession(ctx context.Context, sid string, sess UserSession, ttl time.Duration) error {
	return SetJSON(c, ctx, constants.SetupSessionPrefix+sid, sess, ttl)
}

func (c *Cache) GetSetupSession(ctx context.Context, sid string) (*UserSession, error) {
	return GetJSON[UserSession](c, ctx, constants.SetupSessionPrefix+sid)
}

func (c *Cache) DeleteSetupSession(ctx context.Context, sid string) error {
	return c.Del(ctx, constants.SetupSessionPrefix+sid)
}
