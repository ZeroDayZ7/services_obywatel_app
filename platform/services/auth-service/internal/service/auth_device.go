package service

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/schemas"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/auth-service/internal/http"
)

// #region VerifyDeviceSignature
//
//
// #region VerifyDeviceSignature
func (s *authService) VerifyDeviceSignature(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, signature, fingerprint string) (*http.LoginResponse, error) {
	log := shared.GetLogger()

	device, err := s.userRepo.GetDeviceByFingerprint(ctx, userID, fingerprint)
	if err != nil || device == nil {
		log.WarnMap("Device not found or inactive", map[string]any{"user": userID, "fpt": fingerprint})
		return nil, errors.ErrUntrustedDevice
	}

	pubKeyBytes, err := base64.StdEncoding.DecodeString(device.PublicKey)
	if err != nil {
		log.ErrorObj("Failed to decode device public key", err)
		return nil, errors.ErrInternal
	}

	// Weryfikacja challenge session (odczyt z Redisa, usuwanie i sprawdzanie podpisu Ed25519 z Domain Separator)
	if err := s.verifyChallengeSession(ctx, sessionID, signature, pubKeyBytes); err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return nil, errors.ErrUserNotFound
	}

	if err := s.CanUserLogin(user); err != nil {
		log.WarnMap("User account locked/banned during device verification", map[string]any{
			"user_id": user.ID,
			"status":  user.Status,
		})
		return nil, err
	}

	accessToken, newSessionID, err := s.CreateAccessToken(ctx, user.ID, fingerprint)
	if err != nil {
		return nil, errors.ErrInternal
	}

	refreshToken, err := s.CreateRefreshToken(user.ID, fingerprint, nil)
	if err != nil {
		return nil, errors.ErrInternal
	}

	// Wyciąganie szczegółów profilu pracownika/instytucji
	var empNumber, instID, deptID string
	permissions := []string{}

	if user.EmployeeProfile != nil {
		empNumber = user.EmployeeProfile.EmployeeNumber

		if user.EmployeeProfile.InstitutionID != uuid.Nil {
			instID = user.EmployeeProfile.InstitutionID.String()
		}
		if user.EmployeeProfile.DepartmentID != uuid.Nil {
			deptID = user.EmployeeProfile.DepartmentID.String()
		}
		if user.EmployeeProfile.Permissions != nil {
			permissions = user.EmployeeProfile.Permissions
		}
	}

	// Pełne zestawienie danych sesyjnych zapisywanych w Redis dla Gatewaya
	sessionData := redis.UserSession{
		UserID:         user.ID.String(),
		Username:       user.Username,
		Email:          user.Email,
		Role:           string(user.Role),
		EmployeeNumber: empNumber,
		InstitutionID:  instID,
		DepartmentID:   deptID,
		Permissions:    permissions,
		Fingerprint:    shared.HashSHA256(fingerprint),
		PublicKey:      device.PublicKey,
	}

	if err := s.cache.SetSession(ctx, newSessionID, &sessionData, s.cfg.Session.TTL); err != nil {
		log.ErrorObj("Failed to save session in Redis", err)
		return nil, errors.ErrInternal
	}

	return &http.LoginResponse{
		Type:         "fullSuccess",
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
		UserID:       user.ID.String(),
		ExpiresAt:    time.Now().Add(s.cfg.JWT.AccessTTL).Unix(),
	}, nil
}

// #region UnpairDevice
func (s *authService) UnpairDevice(ctx context.Context, userID uuid.UUID, deviceFingerprint string, sessionID uuid.UUID, req schemas.UnpairDeviceRequest) error {
	log := shared.GetLogger()

	device, err := s.userRepo.GetDeviceByFingerprint(ctx, userID, deviceFingerprint)
	if err != nil || device == nil {
		log.WarnMap("UnpairDevice: device not found", map[string]any{
			"user_id":     userID,
			"fingerprint": deviceFingerprint,
		})
		return errors.ErrUntrustedDevice
	}

	if err := s.userRepo.DeleteDevice(ctx, userID, deviceFingerprint); err != nil {
		log.ErrorObj("UnpairDevice: failed to delete device from DB", err)
		return errors.ErrInternal
	}

	if err := s.refreshRepo.RevokeByFingerprint(ctx, userID, deviceFingerprint); err != nil {
		log.WarnObj("UnpairDevice: failed to revoke refresh tokens", err)
	}

	if sessionID != uuid.Nil {
		if err := s.cache.DeleteSession(ctx, sessionID); err != nil {
			log.WarnObj("UnpairDevice: failed to clear session cache", err)
		}
	}

	return nil
}
