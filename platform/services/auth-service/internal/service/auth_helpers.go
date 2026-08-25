package service

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/constants"
	"github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/security"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/auth-service/internal/model"
)

// #region CanUserLogin
func (s *authService) CanUserLogin(user *model.User) error {
	// 1. Najpierw sprawdzamy statusy stałe
	switch user.Status {
	case model.StatusBanned:
		return errors.ErrAccountBanned
	case model.StatusLocked:
		return errors.ErrAccountLocked
	case model.StatusPending:
		return errors.ErrAccountPending
	}

	// 2. Obsługa StatusSuspended (Blokada czasowa)
	if user.Status == model.StatusSuspended {
		// Sprawdzamy, czy czas blokady już minął
		if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
			// Obliczamy pozostały czas (opcjonalnie do metadanych)
			remaining := time.Until(*user.LockedUntil).Minutes()

			// Zwracamy błąd czasowy z informacją o minutach
			return errors.ErrAccountTemporarilyLocked.WithMeta("remaining_minutes", int(remaining))
		}

		// Jeśli czas blokady minął, pozwalamy na login
		// (Status zostanie zaktualizowany na ACTIVE po poprawnym sprawdzeniu hasła)
		return nil
	}

	// 3. Jeśli status to ACTIVE
	if user.Status == model.StatusActive {
		return nil
	}

	// 4. Fallback dla nieznanych statusów
	return errors.ErrInternal
}

// #region CreateAccessToken
func (s *authService) CreateAccessToken(ctx context.Context, userID uuid.UUID, fingerprint string) (string, string, error) {
	sessionID := shared.GenerateSessionID()
	claims := jwt.MapClaims{
		"uid":   userID,
		"sid":   sessionID,
		"fpt":   shared.HashSHA256(fingerprint),
		"scope": constants.ScopeAccess.String(),
	}

	token, err := security.GenerateJWTViaKMS(ctx, s.cfg.ToKMSServiceConfig(), "shared-jwt", claims, s.cfg.JWT.AccessTTL)
	if err != nil {
		return "", "", fmt.Errorf("auth: failed to generate access token via KMS: %w", err)
	}

	return token, sessionID, nil
}

// #region CreateSetupToken
func (s *authService) CreateSetupToken(ctx context.Context, userID uuid.UUID, fingerprint string) (string, string, error) {
	sessionID := shared.GenerateSessionID()
	claims := jwt.MapClaims{
		"uid":   userID.String(),
		"sid":   sessionID,
		"fpt":   shared.HashSHA256(fingerprint),
		"scope": constants.ScopeDeviceVerify.String(),
	}

	setupTTL := 15 * time.Minute

	token, err := security.GenerateJWTViaKMS(ctx, s.cfg.ToKMSServiceConfig(), "shared-jwt", claims, setupTTL)
	if err != nil {
		return "", "", fmt.Errorf("auth: failed to generate setup token via KMS: %w", err)
	}

	return token, sessionID, nil
}

// #region CreateRefreshToken
func (s *authService) CreateRefreshToken(userID uuid.UUID, fingerprint string, deviceID *uuid.UUID) (*model.RefreshToken, error) {
	rawToken, err := security.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	rt := &model.RefreshToken{
		UserID:            userID,
		DeviceID:          deviceID,
		DeviceFingerprint: shared.HashSHA256(fingerprint),
		Token:             shared.HashSHA256(rawToken),
		ExpiresAt:         time.Now().Add(s.cfg.JWT.RefreshTTL),
		Revoked:           false,
	}

	if err := s.refreshRepo.Save(rt); err != nil {
		return nil, err
	}

	rt.Token = rawToken
	return rt, nil
}

// #region handleFailedLogin
func (s *authService) handleFailedLogin(ctx context.Context, userID uuid.UUID) error {
	log := shared.GetLogger()

	attempts, incErr := s.userRepo.IncrementUserFailedLogin(ctx, userID)
	if incErr != nil {
		log.Error("Failed to increment failed attempts", incErr)
	}

	if attempts >= 5 {
		_ = s.userRepo.PermanentLock(ctx, userID)
		return errors.ErrAccountLocked
	}

	return errors.ErrInvalidCredentials
}

// #region createChallengeSession
func (s *authService) createChallengeSession(ctx context.Context, userID uuid.UUID, fingerprint string) (setupToken string, sessionID string, challenge string, err error) {
	log := shared.GetLogger()

	setupToken, sessionID, err = s.CreateSetupToken(ctx, userID, fingerprint)
	if err != nil {
		log.ErrorObj("Failed to create setup token", err)
		return "", "", "", errors.ErrInternal
	}

	challenge, err = security.GenerateRandomString(32)
	if err != nil {
		log.ErrorObj("Failed to generate challenge", err)
		return "", "", "", errors.ErrInternal
	}

	if err := s.cache.SetChallenge(ctx, sessionID, challenge, 5*time.Minute); err != nil {
		log.ErrorObj("Failed to save challenge in Redis", err)
		return "", "", "", errors.ErrInternal
	}

	return setupToken, sessionID, challenge, nil
}

// #region verifyChallengeSession
func (s *authService) verifyChallengeSession(ctx context.Context, sessionID string, signatureB64 string, pubKeyBytes []byte) error {
	log := shared.GetLogger()

	// 1. Odczyt challenge z Redisa
	storedChallenge, err := s.cache.GetChallenge(ctx, sessionID)
	if err != nil || storedChallenge == "" {
		log.Warn("Challenge session expired or not found in Redis", "sessionID", sessionID)
		return errors.ErrChallengeExpired
	}

	// 2. Bezwzględne usunięcie z Redisa (Atomic Burn - zapobieganie Replay Attack)
	_ = s.cache.DeleteChallenge(ctx, sessionID)

	// 3. Kryptograficzne sprawdzenie podpisu z użyciem funkcji z pkg/security
	domain := s.cfg.Auth.Domain
	if err := security.VerifyEd25519Challenge(pubKeyBytes, storedChallenge, signatureB64, domain); err != nil {
		log.Warn("Cryptographic signature verification failed", "sessionID", sessionID, "error", err)
		return errors.ErrInvalidSignature
	}

	return nil
}
