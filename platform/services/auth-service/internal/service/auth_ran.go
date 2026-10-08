package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/permissions"
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
func (s *authService) Logout(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, fingerprint string) error {
	log := shared.GetLogger()

	// 1. Pobierz sesję
	session, err := s.cache.GetSession(ctx, sessionID)
	if err != nil {
		log.WarnMap("Logout: session not found", map[string]any{"sid": sessionID})
		return errors.ErrUnauthorized
	}

	// 2. Weryfikacja bezpieczeństwa (UserID + normalizacja identyfikatora urządzenia)
	trimmedFingerprint := strings.TrimSpace(fingerprint)
	trimmedSessionFingerprint := strings.TrimSpace(session.Fingerprint)
	trimmedSessionDeviceID := strings.TrimSpace(session.DeviceID)

	matchesDeviceID := trimmedFingerprint != "" &&
		(trimmedSessionFingerprint == trimmedFingerprint || trimmedSessionDeviceID == trimmedFingerprint)
	matchesSessionFingerprint := trimmedSessionFingerprint != "" &&
		(trimmedSessionFingerprint == trimmedFingerprint || trimmedSessionDeviceID == trimmedSessionFingerprint)

	if session.UserID != userID.String() || (!matchesDeviceID && !matchesSessionFingerprint) {
		log.ErrorMap("Logout security violation", map[string]any{
			"expected_uid": userID.String(),
			"actual_uid":   session.UserID,
			"expected_fpt": fingerprint,
			"actual_fpt":   session.Fingerprint,
			"session_dev":  session.DeviceID,
		})
		return errors.ErrUnauthorized
	}

	// 3. Usuwanie sesji z Redis
	if err := s.cache.DeleteSession(ctx, sessionID); err != nil {
		return errors.ErrInternal
	}

	// 4. Unieważnienie Refresh Tokena w DB przy użyciu zgodnego identyfikatora urządzenia
	resolverFingerprint := trimmedSessionFingerprint
	if resolverFingerprint == "" {
		resolverFingerprint = trimmedSessionDeviceID
	}
	if resolverFingerprint == "" {
		resolverFingerprint = trimmedFingerprint
	}
	_ = s.refreshRepo.RevokeByFingerprint(ctx, userID, resolverFingerprint)

	return nil
}
