package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/auth-service/internal/http"
)

// #region RefreshToken
func (s *authService) RefreshToken(ctx context.Context, tokenStr string, fingerprint string) (*http.RefreshResponse, error) {
	log := shared.GetLogger()

	// 1. Hashowanie tokena
	hash := sha256.Sum256([]byte(tokenStr))
	hashedTokenHex := hex.EncodeToString(hash[:])

	// 2. Pobranie i weryfikacja stanu Refresh Tokena
	rt, err := s.refreshRepo.GetByToken(hashedTokenHex)
	if err != nil || rt.Revoked || rt.ExpiresAt.Before(time.Now()) {
		log.WarnObj("Invalid, revoked or expired refresh token", tokenStr)
		return nil, errors.ErrInvalidToken
	}

	// 3. Weryfikacja zgodności fingerprintu z tokenem
	if rt.DeviceFingerprint != fingerprint {
		log.WarnMap("SECURITY ALERT: Refresh token used on different device!", map[string]any{
			"user_id":      rt.UserID,
			"expected_fpt": rt.DeviceFingerprint,
			"received_fpt": fingerprint,
		})
		return nil, errors.ErrInvalidToken
	}

	// 4. WERYFIKACJA ZAUFANEGO URZĄDZENIA (UserDevice)
	device, err := s.userRepo.GetDeviceByFingerprint(ctx, rt.UserID, fingerprint)
	if err != nil || device == nil || !device.IsActive || !device.IsVerified {
		log.WarnMap("SECURITY ALERT: Refresh token used on untrusted, inactive or deleted device!", map[string]any{
			"user_id":     rt.UserID,
			"fingerprint": fingerprint,
		})
		return nil, errors.ErrUntrustedDevice
	}

	// 5. Pobranie użytkownika i weryfikacja statusu konta
	user, err := s.userRepo.GetByID(ctx, rt.UserID)
	if err != nil || user == nil {
		return nil, errors.ErrUserNotFound
	}

	if err := s.CanUserLogin(user); err != nil {
		log.WarnMap("User account locked/banned during token refresh", map[string]any{
			"user_id": user.ID,
			"status":  user.Status,
		})
		return nil, err
	}

	// 6. ROTACJA TOKENÓW: Unieważniamy stary token
	rt.Revoked = true
	if err := s.refreshRepo.Update(rt); err != nil {
		log.ErrorObj("Failed to revoke old refresh token", err)
		return nil, errors.ErrInternal
	}

	// 7. Generowanie NOWYCH tokenów (przypisujemy zweryfikowany device.ID)
	accessToken, newSessionID, err := s.CreateAccessToken(ctx, user.ID, fingerprint)
	if err != nil {
		return nil, errors.ErrInternal
	}

	newRefreshToken, err := s.CreateRefreshToken(user.ID, fingerprint, &device.ID)
	if err != nil {
		return nil, errors.ErrInternal
	}

	// 8. Zapis sesji w Redis
	sessionData := redis.UserSession{
		UserID:      user.ID.String(),
		Fingerprint: shared.HashSHA256(fingerprint),
		PublicKey:   device.PublicKey,
		Role:        string(user.Role),
	}

	if err := s.cache.SetSession(ctx, newSessionID, sessionData, s.cfg.Session.TTL); err != nil {
		log.ErrorObj("Failed to save session in Redis", err)
		return nil, errors.ErrInternal
	}

	return &http.RefreshResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken.Token,
		ExpiresAt:    time.Now().Add(s.cfg.JWT.AccessTTL).Unix(),
	}, nil
}

// region RevokeRefreshToken
// #region RevokeRefreshToken
func (s *authService) RevokeRefreshToken(token string) error {
	rt, err := s.refreshRepo.GetByToken(token)
	if err != nil {
		return err
	}
	rt.Revoked = true
	return s.refreshRepo.Update(rt)
}
