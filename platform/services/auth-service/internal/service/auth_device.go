package service

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/schemas"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/auth-service/internal/http"
	"github.com/zerodayz7/platform/services/auth-service/internal/model"
)

// #region VerifyDeviceSignature
func (s *authService) VerifyDeviceSignature(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, deviceID string, signature string) (*http.LoginResponse, error) {
	log := shared.GetLogger()

	// 1. Pobranie sesji parowania/wyzwania z Redisa
	setupSess, err := s.cache.GetSetupSession(ctx, sessionID)
	if err != nil || setupSess == nil {
		log.WarnMap("[VerifyDeviceSignature] Challenge session expired or not found", map[string]any{
			"session_id": sessionID,
			"err":        err,
		})
		return nil, errors.ErrChallengeExpired
	}

	// 2. Pobranie klucza publicznego zaufanego urządzenia po deviceID (zhashowany fingerprint)
	device, err := s.userRepo.GetDeviceByFingerprint(ctx, userID, deviceID)
	if err != nil || device == nil {
		log.WarnMap("Device not found or inactive", map[string]any{"user": userID, "device_id": deviceID})
		return nil, errors.ErrUntrustedDevice
	}

	pubKeyBytes, err := base64.StdEncoding.DecodeString(device.PublicKey)
	if err != nil {
		log.ErrorObj("Failed to decode device public key", err)
		return nil, errors.ErrInternal
	}

	// 3. Weryfikacja kryptograficzna podpisu
	if err := s.verifyChallengeSession(setupSess.Challenge, signature, pubKeyBytes); err != nil {
		return nil, err
	}

	// Posprzątaj sesję wyzwania po użyciu
	_ = s.cache.DeleteSetupSession(ctx, sessionID)

	// 4. Pobranie użytkownika i weryfikacja stanu konta
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

	// 5. Generowanie docelowych tokenów JWT oraz wpisu sesji w Redis
	accessToken, newSessionID, err := s.CreateAccessToken(ctx, user.ID, deviceID)
	if err != nil {
		return nil, errors.ErrInternal
	}

	refreshToken, err := s.CreateRefreshToken(user.ID, deviceID, &device.ID)
	if err != nil {
		return nil, errors.ErrInternal
	}

	sessionData := s.buildUserSession(user, deviceID, device.PublicKey, false)

	if err := s.cache.SetSession(ctx, newSessionID, &sessionData, s.cfg.Session.TTL); err != nil {
		log.ErrorObj("Failed to save session in Redis", err)
		return nil, errors.ErrInternal
	}

	expiresAt := time.Now().Add(s.cfg.JWT.AccessTTL).Unix()

	return &http.LoginResponse{
		Type: http.LoginResultSuccess,
		Success: &http.LoginSuccessData{
			AccessToken:  accessToken,
			RefreshToken: refreshToken.Token,
			UserID:       user.ID.String(),
			ExpiresAt:    expiresAt,
		},
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

// region RegisterUserDevice
// #region RegisterUserDevice
func (s *authService) RegisterUserDevice(ctx context.Context, userID uuid.UUID, fingerprint, publicKey, deviceName, platform string, isVerified bool, lastIp string) error {
	device := model.UserDevice{
		UserID: userID, DeviceFingerprint: fingerprint, PublicKey: publicKey,
		DeviceNameEncrypted: deviceName, Platform: platform, IsVerified: isVerified,
		LastIP: lastIp, IsActive: true,
	}
	return s.userRepo.SaveDevice(ctx, &device)
}

// #region RegisterDevice
func (s *authService) RegisterDevice(
	ctx context.Context,
	userID uuid.UUID,
	sessionID uuid.UUID,
	deviceID string,
	clientIP string,
	req schemas.RegisterDeviceRequest,
) (*http.RegisterDeviceResponse, error) {
	log := shared.GetLogger()

	if sessionID == uuid.Nil {
		log.WarnMap("[RegisterDevice] Empty sessionID passed", map[string]any{"user_id": userID.String()})
		return nil, errors.ErrSessionExpired
	}

	// 1. Walidacja tymczasowej sesji parowania (Setup Session)
	setupSess, err := s.cache.GetSetupSession(ctx, sessionID)
	if err != nil || setupSess == nil {
		log.WarnMap("[RegisterDevice] Setup session expired or missing", map[string]any{"sid": sessionID})
		return nil, errors.ErrSessionExpired
	}

	// 2. Weryfikacja kryptograficzna wyzwania z pobranej wyżej sesji
	pubKeyBytes, err := base64.StdEncoding.DecodeString(req.PublicKey)
	if err != nil || len(pubKeyBytes) != ed25519.PublicKeySize {
		log.ErrorObj("[RegisterDevice] Invalid public key format", err)
		return nil, errors.ErrInvalidPairingData
	}

	if err := s.verifyChallengeSession(setupSess.Challenge, req.Signature, pubKeyBytes); err != nil {
		log.WarnMap("[RegisterDevice] Cryptographic verification failed", map[string]any{"sid": sessionID, "err": err})
		return nil, err
	}

	// Usunięcie całej sesji setup po pomyślnej weryfikacji
	_ = s.cache.DeleteSetupSession(ctx, sessionID)

	// 3. Pobranie użytkownika i weryfikacja stanu konta
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		log.ErrorObj("[RegisterDevice] User not found", err)
		return nil, errors.ErrUserNotFound
	}

	if err := s.CanUserLogin(user); err != nil {
		log.WarnMap("[RegisterDevice] Account locked or banned", map[string]any{"user_id": user.ID, "status": user.Status})
		return nil, err
	}

	// 4. Utworzenie lub aktualizacja rekordu urządzenia
	device := &model.UserDevice{
		UserID:              userID,
		DeviceFingerprint:   deviceID,
		PublicKey:           req.PublicKey,
		DeviceNameEncrypted: req.DeviceNameEncrypted,
		Platform:            req.Platform,
		IsVerified:          true,
		IsActive:            true,
		LastIP:              clientIP,
	}

	if err := s.userRepo.SaveDevice(ctx, device); err != nil {
		log.ErrorObj("[RegisterDevice] Failed to save device in DB", err)
		return nil, errors.ErrInternal
	}

	// 5. Generowanie poświadczeń (JWT Access & Refresh Token)
	accessToken, newSID, err := s.CreateAccessToken(ctx, userID, deviceID)
	if err != nil {
		return nil, errors.ErrInternal
	}

	refreshToken, err := s.CreateRefreshToken(userID, deviceID, &device.ID)
	if err != nil {
		return nil, errors.ErrInternal
	}

	// 6. Zapis pełnej sesji użytkownika w Redis (dla API Gateway)
	sessionData := s.buildUserSession(user, deviceID, req.PublicKey, false)
	if err := s.cache.SetSession(ctx, newSID, &sessionData, s.cfg.Session.TTL); err != nil {
		log.ErrorObj("[RegisterDevice] Failed to persist session in Redis", err)
		return nil, errors.ErrInternal
	}

	log.InfoMap("✅ Device registered successfully", map[string]any{"user_id": userID.String(), "device_id": device.ID})

	return &http.RegisterDeviceResponse{
		Success:      true,
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
		IsTrusted:    true,
		User: http.DeviceUserData{
			UserID:      user.ID.String(),
			Email:       user.Email,
			DisplayName: user.Username,
			LastLogin:   time.Now().Format(time.RFC3339),
			Roles:       []string{string(user.Role)},
		},
	}, nil
}
