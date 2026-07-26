// cmdr: redis/sessions.go

package redis

import (
	"context"
	"time"
)

type UserSession struct {
	UserID      string   `json:"user_id"`
	Fingerprint string   `json:"fingerprint"`
	Role        string   `json:"role,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	Challenge   string   `json:"challenge,omitempty"`
	IP          string   `json:"ip,omitempty"`
}

// --- Metody dla Sesji Głównej ---

func (c *Cache) SetSession(ctx context.Context, sid string, sess UserSession, ttl time.Duration) error {
	return SetJSON(c, ctx, SessionPrefix+sid, sess, ttl)
}

func (c *Cache) GetSession(ctx context.Context, sid string) (*UserSession, error) {
	return GetJSON[UserSession](c, ctx, SessionPrefix+sid)
}

func (c *Cache) DeleteSession(ctx context.Context, sid string) error {
	return c.Del(ctx, SessionPrefix+sid)
}

func (c *Cache) UpdateSession(ctx context.Context, sid string, updateFn func(*UserSession)) error {
	session, err := c.GetSession(ctx, sid)
	if err != nil {
		return err
	}

	updateFn(session)

	ttl, _ := c.client.TTL(ctx, SessionPrefix+sid).Result()
	if ttl <= 0 {
		ttl = c.ttl
	}

	return c.SetSession(ctx, sid, *session, ttl)
}

// --- Metody dla Challenge (Ed25519) ---

func (c *Cache) SetChallenge(ctx context.Context, sid string, challenge string, ttl time.Duration) error {
	return c.Set(ctx, ChallengePrefix+sid, challenge, ttl)
}

func (c *Cache) GetChallenge(ctx context.Context, sid string) (string, error) {
	return c.Get(ctx, ChallengePrefix+sid)
}

func (c *Cache) DeleteChallenge(ctx context.Context, sid string) error {
	return c.Del(ctx, ChallengePrefix+sid)
}

// --- Metody dla Sesji Tymczasowej (Setup/2FA) ---

func (c *Cache) SetSetupSession(ctx context.Context, sid string, sess UserSession, ttl time.Duration) error {
	return SetJSON(c, ctx, SetupSessionPrefix+sid, sess, ttl)
}

func (c *Cache) GetSetupSession(ctx context.Context, sid string) (*UserSession, error) {
	return GetJSON[UserSession](c, ctx, SetupSessionPrefix+sid)
}

func (c *Cache) DeleteSetupSession(ctx context.Context, sid string) error {
	return c.Del(ctx, SetupSessionPrefix+sid)
}
