package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/permissions"
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/security"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/auth-service/internal/model"
)

// region UpdatePassword
// #region UpdatePassword
func (s *authService) UpdatePassword(ctx context.Context, userID uuid.UUID, newPassword string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return errors.ErrUserNotFound
	}

	passBytes := []byte(newPassword)
	defer clear(passBytes)

	hashed, err := security.HashPassword(passBytes, nil)
	if err != nil {
		return errors.ErrInternal
	}

	now := time.Now()
	user.Password = hashed
	user.PasswordChangedAt = &now

	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	return nil
}

// region Register
// #region Register
func (s *authService) Register(username, email, rawPassword string) (*model.User, error) {
	passBytes := []byte(rawPassword)
	defer clear(passBytes)

	hash, err := security.HashPassword(passBytes, nil)
	if err != nil {
		return nil, errors.ErrInternal
	}

	now := time.Now()
	u := &model.User{
		Username:          username,
		Email:             email,
		Password:          hash,
		PasswordChangedAt: &now,
	}

	// Assign default citizen permissions when registering through public API
	u.Permissions = permissions.DefaultCitizenPermissions

	if err := s.userRepo.CreateUser(u); err != nil {
		return nil, err
	}

	return u, nil
}

// #region Logout
func (s *authService) Logout(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, deviceID string, refreshToken string) error {
	log := shared.GetLogger()
	trimmedDeviceID := strings.TrimSpace(deviceID)

	var session *redis.UserSession
	if s.cache != nil {
		session, _ = s.cache.GetSession(ctx, sessionID)
		if session != nil {
			if session.UserID != userID.String() {
				log.ErrorMap("Logout security violation: user mismatch", map[string]any{
					"expected_uid": userID.String(),
					"actual_uid":   session.UserID,
					"session_id":   sessionID,
				})
				return errors.ErrUnauthorized
			}

			if trimmedDeviceID != "" {
				trimmedSessionFingerprint := strings.TrimSpace(session.Fingerprint)
				trimmedSessionDeviceID := strings.TrimSpace(session.DeviceID)
				matchesDevice := trimmedSessionFingerprint == trimmedDeviceID || trimmedSessionDeviceID == trimmedDeviceID
				if !matchesDevice {
					log.ErrorMap("Logout security violation: device mismatch", map[string]any{
						"user_id":       userID,
						"session_id":    sessionID,
						"expected_dev":  trimmedDeviceID,
						"actual_fpt":    trimmedSessionFingerprint,
						"actual_dev_id": trimmedSessionDeviceID,
					})
					return errors.ErrUnauthorized
				}
			}
		}
	}

	if refreshToken != "" {
		tokenHash := hashRefreshTokenForLogout(refreshToken)
		if err := s.refreshRepo.Revoke(tokenHash); err != nil {
			log.WarnMap("Logout: refresh token revoke failed", map[string]any{
				"user_id":      userID,
				"session_id":   sessionID,
				"device_id":    trimmedDeviceID,
				"refresh_hash": tokenHash,
				"err":          err,
			})
		}
	}

	if err := s.refreshRepo.RevokeSession(ctx, userID, sessionID); err != nil {
		log.WarnMap("Logout: failed to revoke session in DB", map[string]any{
			"user_id":    userID,
			"session_id": sessionID,
			"device_id":  trimmedDeviceID,
			"err":        err,
		})
	}

	if trimmedDeviceID != "" {
		if err := s.refreshRepo.RevokeByFingerprint(ctx, userID, trimmedDeviceID); err != nil {
			log.WarnMap("Logout: failed to revoke device refresh tokens", map[string]any{
				"user_id":   userID,
				"session_id": sessionID,
				"device_id": trimmedDeviceID,
				"err":       err,
			})
		}
	}

	if s.cache != nil {
		if err := s.cache.DeleteSession(ctx, sessionID); err != nil {
			log.WarnMap("Logout: failed to delete Redis session", map[string]any{
				"user_id":    userID,
				"session_id": sessionID,
				"device_id":  trimmedDeviceID,
				"err":        err,
			})
		}
		_ = s.cache.DeleteSetupSession(ctx, sessionID)
	}

	clientIP := ""
	if session != nil {
		clientIP = session.IP
	}
	log.InfoMap("Logout successful", map[string]any{
		"user_id":    userID,
		"session_id": sessionID,
		"device_id":  trimmedDeviceID,
		"client_ip":  clientIP,
		"refresh_req": refreshToken != "",
	})

	return nil
}
