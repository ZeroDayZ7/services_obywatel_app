// cmdr: redis/auth_2fa.go

package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/zerodayz7/platform/pkg/constants"
)

type TwoFASession struct {
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	CodeHash    string `json:"code_hash"`
	Token       string `json:"token"`
	Fingerprint string `json:"fingerprint"`
	Attempts    int    `json:"attempts"`
}

//#region Set2FASession
func (c *Cache) Set2FASession(ctx context.Context, token string, sess TwoFASession, ttl time.Duration) error {
	return SetJSON(c, ctx, constants.Login2FAPrefix+token, sess, ttl)
}

//#region Get2FASession
func (c *Cache) Get2FASession(ctx context.Context, token string) (*TwoFASession, error) {
	return GetJSON[TwoFASession](c, ctx, constants.Login2FAPrefix+token)
}

//#region Delete2FASession
func (c *Cache) Delete2FASession(ctx context.Context, token string) error {
	return c.Del(ctx, constants.Login2FAPrefix+token)
}

//#region Verify2FAAttempt
func (c *Cache) Verify2FAAttempt(
	ctx context.Context,
	token string,
	maxAttempts int,
	ttl time.Duration,
) (string, error) {
	fullKey := constants.Login2FAPrefix + token

	res, err := c.client.Eval(
		ctx,
		verify2FAScript,
		[]string{fullKey},
		maxAttempts,
		int(ttl.Seconds()),
	).Result()
	if err != nil {
		return "", fmt.Errorf("lua execution failed: %w", err)
	}

	arr, ok := res.([]interface{})
	if !ok || len(arr) == 0 {
		return "", errors.New("invalid lua response format from verify2fa script")
	}

	status, ok := arr[0].(string)
	if !ok {
		return "", errors.New("invalid status type in lua response")
	}

	switch status {
	case "NOT_FOUND":
		return "not_found", nil
	case "LOCKED":
		return "locked", nil
	case "ATTEMPT_UPDATED":
		return "attempt_updated", nil
	default:
		return "", fmt.Errorf("unknown 2FA status from redis lua script: %s", status)
	}
}
