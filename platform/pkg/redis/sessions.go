package redis

import (
	"context"
	"encoding/json"
	"time"
)

// UserSession przechowuje dane aktywnej sesji użytkownika
type UserSession struct {
	UserID      string   `json:"user_id"`
	Fingerprint string   `json:"fingerprint"`
	Roles       []string `json:"roles,omitempty"`
	Challenge   string   `json:"challenge,omitempty"`
	IP          string   `json:"ip,omitempty"`
}

// --- Metody dla Sesji Głównej ---

// SetSession zapisuje sesję użytkownika z odpowiednim prefixem
func (c *Cache) SetSession(ctx context.Context, sid string, sess UserSession, ttl time.Duration) error {
	data, _ := json.Marshal(sess)
	return c.client.Set(ctx, SessionPrefix+sid, data, ttl).Err()
}

// GetSession pobiera i deserializuje sesję
func (c *Cache) GetSession(ctx context.Context, sid string) (*UserSession, error) {
	data, err := c.client.Get(ctx, SessionPrefix+sid).Result()
	if err != nil {
		return nil, err
	}
	var sess UserSession
	if err := json.Unmarshal([]byte(data), &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

// DeleteSession usuwa sesję użytkownika
func (c *Cache) DeleteSession(ctx context.Context, sid string) error {
	return c.client.Del(ctx, SessionPrefix+sid).Err()
}

// UpdateSession pozwala na atomową modyfikację sesji za pomocą funkcji
func (c *Cache) UpdateSession(ctx context.Context, sid string, updateFn func(*UserSession)) error {
	session, err := c.GetSession(ctx, sid)
	if err != nil {
		return err
	}

	updateFn(session)

	// Pobieramy pozostały czas życia klucza, aby nie resetować go przy aktualizacji
	ttl, _ := c.client.TTL(ctx, SessionPrefix+sid).Result()
	if ttl <= 0 {
		ttl = c.ttl
	}

	return c.SetSession(ctx, sid, *session, ttl)
}

// --- Metody dla Challenge (Ed25519) ---

// SetChallenge zapisuje wyzwanie kryptograficzne
func (c *Cache) SetChallenge(ctx context.Context, sid string, challenge string, ttl time.Duration) error {
	return c.client.Set(ctx, ChallengePrefix+sid, challenge, ttl).Err()
}

// GetChallenge pobiera wyzwanie dla danej sesji
func (c *Cache) GetChallenge(ctx context.Context, sid string) (string, error) {
	return c.client.Get(ctx, ChallengePrefix+sid).Result()
}

// DeleteChallenge usuwa wyzwanie po poprawnej weryfikacji (zabezpieczenie Replay Attack)
func (c *Cache) DeleteChallenge(ctx context.Context, sid string) error {
	return c.client.Del(ctx, ChallengePrefix+sid).Err()
}


// --- Metody dla Sesji Tymczasowej (Setup/2FA) ---

// SetSetupSession zapisuje sesję tymczasową (używaną między 2FA a RegisterDevice)
func (c *Cache) SetSetupSession(ctx context.Context, sid string, sess UserSession, ttl time.Duration) error {
    data, _ := json.Marshal(sess)
    return c.client.Set(ctx, SetupSessionPrefix+sid, data, ttl).Err()
}

// GetSetupSession pobiera sesję tymczasową
func (c *Cache) GetSetupSession(ctx context.Context, sid string) (*UserSession, error) {
    data, err := c.client.Get(ctx, SetupSessionPrefix+sid).Result()
    if err != nil {
        return nil, err
    }
    var sess UserSession
    if err := json.Unmarshal([]byte(data), &sess); err != nil {
        return nil, err
    }
    return &sess, nil
}

// DeleteSetupSession usuwa sesję tymczasową po jej "awansowaniu" na pełną sesję
func (c *Cache) DeleteSetupSession(ctx context.Context, sid string) error {
    return c.client.Del(ctx, SetupSessionPrefix+sid).Err()
}
