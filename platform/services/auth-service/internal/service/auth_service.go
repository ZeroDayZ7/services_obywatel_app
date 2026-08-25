package service

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/constants"
	"github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/schemas"
	"github.com/zerodayz7/platform/pkg/security"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/auth-service/config"
	"github.com/zerodayz7/platform/services/auth-service/internal/http"
	"github.com/zerodayz7/platform/services/auth-service/internal/model"
	repo "github.com/zerodayz7/platform/services/auth-service/internal/repository"
)

// region interface
type AuthService interface {
	AttemptLogin(ctx context.Context, email string, password []byte, fingerprint string) (*http.LoginResponse, error)
	Register(username, email, rawPassword string) (*model.User, error)
	UpdatePassword(ctx context.Context, userID uuid.UUID, newPassword string) error
	Verify2FA(ctx context.Context, token string, code []byte, fingerprint string, ip string) (*http.Verify2FAResponse, error)
	Resend2FACode(ctx context.Context, email string, token string) error
	Logout(ctx context.Context, userID uuid.UUID, sessionID string, fingerprint string) error
	RegisterDevice(ctx context.Context, userID uuid.UUID, sessionID string, clientIP string, req schemas.RegisterDeviceRequest) (*http.RegisterDeviceResponse, error)
	RefreshToken(ctx context.Context, tokenStr string, fingerprint string) (*http.RefreshResponse, error)
	VerifyDeviceSignature(ctx context.Context, userID, challenge, signature, fingerprint string) (*http.LoginResponse, error)
	UnpairDevice(ctx context.Context, userID uuid.UUID, deviceFingerprint, sessionID string, req schemas.UnpairDeviceRequest) error

	AttemptLoginStep2(ctx context.Context, userIDStr, challenge, signature, fingerprint, clientIP string) (*http.LoginResponse, error)
}

// region struct
type authService struct {
	// Zmiana: używaj interfejsu z repozytorium
	userRepo     repo.UserRepository
	employeeRepo repo.EmployeeRepository
	refreshRepo  repo.RefreshTokenRepository
	cache        *redis.Cache
	cfg          *config.Config
}

// #region NewAuthService
func NewAuthService(
	userRepo repo.UserRepository,
	employeeRepo repo.EmployeeRepository,
	refreshRepo repo.RefreshTokenRepository,
	cache *redis.Cache,
	cfg *config.Config,
) AuthService {
	return &authService{
		userRepo:     userRepo,
		employeeRepo: employeeRepo,
		refreshRepo:  refreshRepo,
		cache:        cache,
		cfg:          cfg,
	}
}

// #region AttemptLoginStep2
func (s *authService) AttemptLoginStep2(ctx context.Context, userIDStr string, sessionID string, signature string, fingerprint string, clientIP string) (*http.LoginResponse, error) {
	log := shared.GetLogger()

	log.InfoMap("[AttemptLoginStep2] Rozpoczęcie weryfikacji Step 2", map[string]any{
		"user_id_input": userIDStr,
		"session_id":    sessionID,
		"signature_len": len(signature),
		"client_ip":     clientIP,
	})

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.WarnMap("[AttemptLoginStep2] Błąd parsowania UUID użytkownika", map[string]any{
			"user_id_input": userIDStr,
			"err":           err.Error(),
		})
		return nil, errors.ErrInvalidParams
	}

	// 1. Pobranie wyzwania z Redisa na podstawie SessionID
	storedChallenge, err := s.cache.GetChallenge(ctx, sessionID)
	if err != nil || storedChallenge == "" {
		log.WarnMap("[AttemptLoginStep2] Challenge nie znaleziony w Redis lub wygasł", map[string]any{
			"user_id": userIDStr,
			"sid":     sessionID,
			"err":     err,
		})
		return nil, errors.ErrInvalidChallenge
	}

	log.InfoMap("[AttemptLoginStep2] Odczytano challenge z Redis", map[string]any{
		"user_id":          userIDStr,
		"stored_challenge": storedChallenge,
	})

	// 2. Pobranie aktywnego poświadczenia (karty/klucza) pracownika
	cred, err := s.employeeRepo.GetActiveCredentialByUserID(ctx, userID)
	if err != nil || cred == nil {
		log.WarnMap("[AttemptLoginStep2] Nie znaleziono aktywnej karty/poświadczenia w DB", map[string]any{
			"user_id": userIDStr,
			"err":     err,
		})
		return nil, errors.ErrUntrustedDevice
	}

	log.InfoMap("[AttemptLoginStep2] Znaleziono poświadczenie pracownika w DB", map[string]any{
		"user_id":    userIDStr,
		"card_sn":    cred.CardSerialNumber,
		"public_key": cred.PublicKey,
		"status":     cred.Status,
	})

	// Sprawdzenie wygaśnięcia karty
	if cred.ExpiresAt != nil && cred.ExpiresAt.Before(time.Now()) {
		log.WarnMap("[AttemptLoginStep2] Karta pracownika wygasła", map[string]any{
			"user_id":    userIDStr,
			"card_sn":    cred.CardSerialNumber,
			"expires_at": cred.ExpiresAt,
		})
		return nil, errors.ErrUntrustedDevice
	}

	// 3. Przygotowanie bajtów wyzwania bezpośrednio jako ciąg UTF-8 (bez dekodowania Base64)
	challengeBytes := []byte(storedChallenge)
	log.InfoMap("[AttemptLoginStep2] Challenge przygotowany jako surowy UTF-8", map[string]any{
		"challenge_bytes_len": len(challengeBytes),
	})

	// 4. Weryfikacja podpisu z klucza publicznego przypisanego do karty pracownika
	log.InfoMap("[AttemptLoginStep2] Próba weryfikacji podpisu Ed25519", map[string]any{
		"pub_key_raw":  cred.PublicKey,
		"signature_in": signature,
	})

	isValid := shared.VerifyEd25519SignatureHex(cred.PublicKey, challengeBytes, signature)
	if !isValid {
		log.WarnMap("SECURITY ALERT: Nieprawidłowy podpis pracownika (VerifyEd25519Signature returned false)", map[string]any{
			"user_id":       userIDStr,
			"card_sn":       cred.CardSerialNumber,
			"pub_key_in_db": cred.PublicKey,
			"signature_in":  signature,
			"challenge_str": storedChallenge,
		})
		return nil, errors.ErrInvalidSignature
	}

	log.Info("[AttemptLoginStep2] Weryfikacja podpisu kryptograficznego zakończona sukcesem!")

	// 5. Usunięcie wyzwania z Redisa (ochrona przed Replay Attack)
	_ = s.cache.DeleteChallenge(ctx, sessionID)

	// 6. Pobranie danych użytkownika oraz profilu pracownika (dla instytucji/uprawnień)
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		log.WarnMap("[AttemptLoginStep2] Nie znaleziono konta użytkownika", map[string]any{
			"user_id": userIDStr,
			"err":     err,
		})
		return nil, errors.ErrUserNotFound
	}

	if err := s.CanUserLogin(user); err != nil {
		log.WarnMap("[AttemptLoginStep2] Konto pracownika jest zablokowane/nieaktywne", map[string]any{
			"user_id": user.ID,
			"err":     err,
		})
		return nil, err
	}

	// Pobieramy dodatkowe metadane urzędnika (Permissions, InstitutionID, DepartmentID)
	var permissions []string
	var instID, deptID, empNumber string

	if user.Role != model.RoleCitizen && user.Role != model.RoleUser {
		empProfile, err := s.employeeRepo.GetProfileByUserID(ctx, user.ID)
		if err == nil && empProfile != nil {
			permissions = empProfile.Permissions
			instID = empProfile.InstitutionID.String()
			deptID = empProfile.DepartmentID.String()
		} else {
			log.WarnMap("[AttemptLoginStep2] Nie pobrano profilu pracownika", map[string]any{
				"user_id": user.ID,
				"err":     err,
			})
		}
	}

	// 7. Generowanie tokenów oraz PEŁNEJ sesji dla Redisa
	accessToken, newSessionID, err := s.CreateAccessToken(ctx, user.ID, fingerprint)
	if err != nil {
		log.ErrorObj("[AttemptLoginStep2] Błąd podczas tworzenia access tokena", err)
		return nil, errors.ErrInternal
	}

	refreshToken, err := s.CreateRefreshToken(user.ID, fingerprint, nil)
	if err != nil {
		log.ErrorObj("[AttemptLoginStep2] Błąd podczas tworzenia refresh tokena", err)
		return nil, errors.ErrInternal
	}

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
		PublicKey:      cred.PublicKey,
	}

	if err := s.cache.SetSession(ctx, newSessionID, sessionData, s.cfg.Session.TTL); err != nil {
		log.ErrorObj("[AttemptLoginStep2] Błąd zapisu sesji w Redis", err)
		return nil, errors.ErrInternal
	}

	log.InfoMap("[AttemptLoginStep2] Logowanie zakończone pełnym sukcesem i zapisem do Redis", map[string]any{
		"user_id":         user.ID.String(),
		"permissions_cnt": len(permissions),
	})

	return &http.LoginResponse{
		Type:         "fullSuccess",
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
		UserID:       user.ID.String(),
		ExpiresAt:    time.Now().Add(s.cfg.JWT.AccessTTL).Unix(),
	}, nil
}

// region UnpairDevice
// #region UnpairDevice
func (s *authService) UnpairDevice(ctx context.Context, userID uuid.UUID, deviceFingerprint, sessionID string, req schemas.UnpairDeviceRequest) error {
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

	if sessionID != "" {
		if err := s.cache.DeleteSession(ctx, sessionID); err != nil {
			log.WarnObj("UnpairDevice: failed to clear session cache", err)
		}
	}

	return nil
}

// region VerifyDeviceSignature
// #region VerifyDeviceSignature
// #region VerifyDeviceSignature
func (s *authService) VerifyDeviceSignature(ctx context.Context, userIDStr, sessionID, signature, fingerprint string) (*http.LoginResponse, error) {
	log := shared.GetLogger()

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, errors.ErrInvalidParams
	}

	// Użycie spójnej metody wrappera z Redisa
	storedChallenge, err := s.cache.GetChallenge(ctx, sessionID)
	if err != nil || storedChallenge == "" {
		log.WarnMap("Challenge not found or expired", map[string]any{"sid": sessionID})
		return nil, errors.ErrInvalidChallenge
	}

	device, err := s.userRepo.GetDeviceByFingerprint(ctx, userID, fingerprint)
	if err != nil || device == nil {
		log.WarnMap("Device not found or inactive", map[string]any{"user": userIDStr, "fpt": fingerprint})
		return nil, errors.ErrUntrustedDevice
	}

	// Dekodowanie wyzwania z Base64 do surowych bajtów binarnych
	challengeBytes, err := decodeChallenge(storedChallenge)
	if err != nil {
		log.ErrorObj("Failed to decode challenge from Base64", err)
		return nil, errors.ErrInvalidParams
	}

	pubKeyBytes, err := base64.StdEncoding.DecodeString(device.PublicKey)
	if err != nil {
		return nil, errors.ErrInternal
	}

	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return nil, errors.ErrInvalidSignature
	}

	if !ed25519.Verify(pubKeyBytes, challengeBytes, sigBytes) {
		log.WarnMap("SECURITY ALERT: Signature mismatch", map[string]any{"userId": userIDStr})
		return nil, errors.ErrInvalidSignature
	}

	// Usuwamy użyty challenge z Redis, aby zapobiec Replay Attacks
	_ = s.cache.DeleteChallenge(ctx, sessionID)

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

	if err := s.cache.SetSession(ctx, newSessionID, sessionData, s.cfg.Session.TTL); err != nil {
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

// region RefreshToken
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

// #region RegisterDevice
func (s *authService) RegisterDevice(ctx context.Context, userID uuid.UUID, sessionID string, clientIP string, req schemas.RegisterDeviceRequest) (*http.RegisterDeviceResponse, error) {
	log := shared.GetLogger()

	log.DebugMap("=== [RegisterDevice] START ===", map[string]any{
		"user_id":            userID.String(),
		"session_id":         sessionID,
		"client_ip":          clientIP,
		"device_fingerprint": req.DeviceFingerprint,
		"platform":           req.Platform,
		"encrypted_name":     req.DeviceNameEncrypted,
		"pubkey_len":         len(req.PublicKey),
		"signature_len":      len(req.Signature),
	})

	// 1. WERYFIKACJA LOGIKI BIZNESOWEJ I SESJI SETUP (NAJPIERW, PRZED ZAPISEM W DB)
	if sessionID != "" {
		log.DebugMap("[RegisterDevice] Checking setup session in Redis", map[string]any{"sid": sessionID})
		setupSess, sessErr := s.cache.GetSetupSession(ctx, sessionID)
		if sessErr != nil || setupSess == nil {
			log.WarnMap("[RegisterDevice] Setup session not found or expired", map[string]any{
				"sid": sessionID,
				"err": sessErr,
			})
			return nil, errors.ErrSessionExpired
		}

		log.DebugMap("[RegisterDevice] Setup session fetched successfully", map[string]any{
			"sid":         sessionID,
			"expected_fp": setupSess.Fingerprint,
			"received_fp": req.DeviceFingerprint,
		})

		hashedFpt := shared.HashSHA256(req.DeviceFingerprint)

		if setupSess.Fingerprint != hashedFpt {
			log.WarnMap("[RegisterDevice] Fingerprint mismatch during registration", map[string]any{
				"expected": setupSess.Fingerprint,
				"received": req.DeviceFingerprint,
				"end":      hashedFpt,
			})
			return nil, errors.ErrInvalidDeviceFingerprint
		}
	} else {
		log.WarnMap("[RegisterDevice] Empty sessionID passed to RegisterDevice", map[string]any{
			"user_id": userID.String(),
		})
	}

	// 2. WERYFIKACJA KRYPTOGRAFICZNA (Challenge / ED25519)
	log.DebugMap("[RegisterDevice] Fetching challenge from Redis", map[string]any{"sid": sessionID})
	storedChallenge, err := s.cache.GetChallenge(ctx, sessionID)
	if err != nil || storedChallenge == "" {
		log.WarnMap("[RegisterDevice] Challenge expired or not found", map[string]any{
			"user_id": userID,
			"sid":     sessionID,
			"err":     err,
		})
		return nil, errors.ErrSessionExpired
	}

	log.DebugMap("[RegisterDevice] Stored challenge retrieved", map[string]any{
		"sid":               sessionID,
		"challenge_raw_len": len(storedChallenge),
		"challenge_raw":     storedChallenge,
	})

	// Dekodowanie challenge z Base64 (jeśli zapisany w Base64) lub bezpośrednia konwersja na bajty
	challengeBytes, err := decodeChallenge(storedChallenge)
	if err != nil {
		log.ErrorMap("[RegisterDevice] Failed to decode challenge Base64", map[string]any{
			"challenge": storedChallenge,
			"err":       err,
		})
		return nil, errors.ErrInvalidPairingData
	}

	log.DebugMap("[RegisterDevice] Decoding Public Key from Base64", map[string]any{
		"pubkey_base64": req.PublicKey,
	})
	pubKeyBytes, err := base64.StdEncoding.DecodeString(req.PublicKey)
	if err != nil {
		log.ErrorObj("[RegisterDevice] Failed to decode public key Base64", err)
		return nil, errors.ErrInvalidPairingData
	}
	if len(pubKeyBytes) != ed25519.PublicKeySize {
		log.ErrorMap("[RegisterDevice] Invalid Ed25519 public key size", map[string]any{
			"expected": ed25519.PublicKeySize,
			"got":      len(pubKeyBytes),
		})
		return nil, errors.ErrInvalidPairingData
	}

	log.DebugMap("[RegisterDevice] Decoding Signature from Base64", map[string]any{
		"signature_base64": req.Signature,
	})
	sigBytes, err := base64.StdEncoding.DecodeString(req.Signature)
	if err != nil {
		log.ErrorObj("[RegisterDevice] Failed to decode signature Base64", err)
		return nil, errors.ErrInvalidPairingData
	}
	if len(sigBytes) != ed25519.SignatureSize {
		log.ErrorMap("[RegisterDevice] Invalid Ed25519 signature size", map[string]any{
			"expected": ed25519.SignatureSize,
			"got":      len(sigBytes),
		})
		return nil, errors.ErrInvalidPairingData
	}

	log.DebugMap("[RegisterDevice] Executing ed25519.Verify check", map[string]any{
		"pubkey_bytes_len":    len(pubKeyBytes),
		"challenge_bytes_len": len(challengeBytes),
		"sig_bytes_len":       len(sigBytes),
	})

	if !ed25519.Verify(pubKeyBytes, challengeBytes, sigBytes) {
		log.ErrorMap("[RegisterDevice] Kryptograficzna weryfikacja urządzenia nieudana", map[string]any{
			"user_id":          userID,
			"sid":              sessionID,
			"challenge_string": storedChallenge,
			"pubkey_base64":    req.PublicKey,
			"sig_base64":       req.Signature,
		})
		return nil, errors.ErrVerificationFailed
	}
	log.DebugMap("[RegisterDevice] ✅ Kryptograficzna weryfikacja ed25519 udana!", map[string]any{
		"user_id": userID.String(),
	})

	// 3. POBRANIE PEŁNYCH DANYCH UŻYTKOWNIKA I WERYFIKACJA STATUSU KONTA
	log.DebugMap("[RegisterDevice] Fetching user record from DB", map[string]any{"user_id": userID.String()})
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		log.ErrorObj("[RegisterDevice] User not found in DB", err)
		return nil, errors.ErrUserNotFound
	}

	if err := s.CanUserLogin(user); err != nil {
		log.WarnMap("[RegisterDevice] User account locked/banned during device registration", map[string]any{
			"user_id": user.ID,
			"status":  user.Status,
			"err":     err,
		})
		return nil, err
	}

	// 4. ZAPIS LUB AKTUALIZACJA URZĄDZENIA W BAZIE DANYCH
	device := &model.UserDevice{
		UserID:              userID,
		DeviceFingerprint:   shared.HashSHA256(req.DeviceFingerprint),
		PublicKey:           req.PublicKey,
		DeviceNameEncrypted: req.DeviceNameEncrypted,
		Platform:            req.Platform,
		IsVerified:          true,
		IsActive:            true,
		LastIP:              clientIP,
	}

	log.DebugMap("[RegisterDevice] Saving device to DB", map[string]any{
		"user_id":     userID.String(),
		"fingerprint": req.DeviceFingerprint,
		"platform":    req.Platform,
	})
	if err := s.userRepo.SaveDevice(ctx, device); err != nil {
		log.ErrorObj("[RegisterDevice] Failed to save device to DB", err)
		return nil, errors.ErrInternal
	}
	log.DebugMap("[RegisterDevice] Device saved successfully", map[string]any{"device_id": device.ID})

	// Posprzątaj sesję tymczasową po pomyślnym zapisie w DB
	if sessionID != "" {
		_ = s.cache.DeleteChallenge(ctx, sessionID)
		_ = s.cache.DeleteSetupSession(ctx, sessionID)
		log.DebugInfo("[RegisterDevice] Setup session and challenge cleared from Redis", sessionID)
	}

	// 5. GENEROWANIE POŚWIADCZEŃ (Z przypisanym DeviceID do RefreshTokena)
	log.DebugMap("[RegisterDevice] Generating AccessToken", map[string]any{"user_id": userID.String()})
	accessToken, newSID, err := s.CreateAccessToken(ctx, userID, req.DeviceFingerprint)
	if err != nil {
		log.ErrorObj("[RegisterDevice] Failed to create access token", err)
		return nil, errors.ErrInternal
	}

	log.DebugMap("[RegisterDevice] Generating RefreshToken", map[string]any{"device_id": device.ID})
	refreshToken, err := s.CreateRefreshToken(userID, req.DeviceFingerprint, &device.ID)
	if err != nil {
		log.ErrorObj("[RegisterDevice] Failed to create refresh token", err)
		return nil, errors.ErrInternal
	}

	// 6. ZAPIS SESJI W REDIS (Pobranie pełnego kontekstu dla Gatewaya)
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

	sessionData := redis.UserSession{
		UserID:         user.ID.String(),
		Username:       user.Username,
		Email:          user.Email,
		Role:           string(user.Role),
		EmployeeNumber: empNumber,
		InstitutionID:  instID,
		DepartmentID:   deptID,
		Permissions:    permissions,
		Fingerprint:    shared.HashSHA256(req.DeviceFingerprint),
		PublicKey:      req.PublicKey,
	}

	log.DebugMap("[RegisterDevice] Persisting new full session to Redis", map[string]any{
		"new_sid": newSID,
		"ttl":     s.cfg.Session.TTL,
	})
	if err = s.cache.SetSession(ctx, newSID, sessionData, s.cfg.Session.TTL); err != nil {
		log.ErrorObj("[RegisterDevice] Failed to save full session to Redis", err)
		return nil, errors.ErrInternal
	}

	log.DebugMap("=== [RegisterDevice] FINISHED SUCCESSFULLY ===", map[string]any{
		"user_id": userID.String(),
		"new_sid": newSID,
	})

	// 7. FINALIZACJA
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

// region Logout
// #region Logout
func (s *authService) Logout(ctx context.Context, userID uuid.UUID, sessionID string, fingerprint string) error {
	log := shared.GetLogger()

	// 1. Pobierz sesję
	session, err := s.cache.GetSession(ctx, sessionID)
	if err != nil {
		log.WarnMap("Logout: session not found", map[string]any{"sid": sessionID})
		return errors.ErrUnauthorized
	}

	// 2. Weryfikacja bezpieczeństwa (UserID i opcjonalnie Fingerprint)
	if session.UserID != userID.String() || session.Fingerprint != fingerprint {
		log.ErrorMap("Logout security violation", map[string]any{
			"expected_uid": userID.String(),
			"actual_uid":   session.UserID,
			"expected_fpt": fingerprint,
			"actual_fpt":   session.Fingerprint,
		})
		return errors.ErrUnauthorized
	}

	// 3. Usuwanie sesji z Redis
	if err := s.cache.DeleteSession(ctx, sessionID); err != nil {
		return errors.ErrInternal
	}

	// 4. Unieważnienie Refresh Tokena w DB przy użyciu fingerprintu
	_ = s.refreshRepo.RevokeByFingerprint(ctx, userID, fingerprint)

	return nil
}

// region Verify2FA
// #region Verify2FA
func (s *authService) Verify2FA(ctx context.Context, token string, code []byte, fingerprint string, ip string) (*http.Verify2FAResponse, error) {
	log := shared.GetLogger()

	// Czyszczenie wrażliwego bufora kodu po zakończeniu funkcji
	defer clear(code)

	// 1. Pobieranie sesji 2FA z Cache
	session, err := s.cache.Get2FASession(ctx, token)
	if err != nil {
		return nil, errors.ErrInvalidCredentials
	}

	log.DebugInfo("2FA compare", map[string]any{
		"hash": session.CodeHash,
	})

	// 2. Weryfikacja kodu Argon2id z uwzględnieniem peppera
	valid, err := security.VerifyPassword(code, session.CodeHash, nil) // podmień s.pepper na nil jeśli nie masz pola w strukturze

	if err != nil || !valid {
		// Logika blokowania po błędnych próbach
		status, _ := s.cache.Verify2FAAttempt(ctx, token, 5, 5*time.Minute)
		log.DebugInfo("2FA verification failed", map[string]any{
			"status": status,
			"token":  token,
		})

		switch status {
		case "locked":
			return nil, errors.Err2FALocked
		default:
			return nil, errors.ErrInvalid2FACode
		}
	}

	// 3. Czyszczenie sesji 2FA
	_ = s.cache.Delete2FASession(ctx, token)

	// 4. Pobieranie użytkownika i aktualizacja metadanych logowania
	uid, _ := uuid.Parse(session.UserID)
	user, err := s.userRepo.GetByID(ctx, uid)
	if err != nil {
		return nil, errors.ErrInternal
	}

	user.LastLogin = time.Now()
	user.LastIP = ip
	_ = s.userRepo.Update(ctx, user)

	setupToken, sessionID, err := s.CreateSetupToken(ctx, uid, fingerprint)
	if err != nil {
		return nil, errors.ErrInternal
	}

	err = s.cache.SetSetupSession(ctx, sessionID, redis.UserSession{
		UserID:      session.UserID,
		Fingerprint: shared.HashSHA256(fingerprint),
	}, s.cfg.Session.TTL)
	if err != nil {
		return nil, errors.ErrInternal
	}

	// 6. Generowanie Challenge (Ed25519)
	challenge, err := security.GenerateRandomString(32)
	if err != nil {
		log.ErrorObj("Failed to generate secure challenge", err)
		return nil, errors.ErrInternal
	}

	if err := s.cache.SetChallenge(ctx, sessionID, challenge, 5*time.Minute); err != nil {
		log.ErrorObj("Failed to save challenge in Redis", err)
		return nil, errors.ErrInternal
	}

	response := &http.Verify2FAResponse{
		Success:    true,
		SetupToken: setupToken,
		Challenge:  challenge,
		IsTrusted:  false,
	}

	// DEBUG INFO: Wypisujemy dokładnie to, co idzie do klienta
	log.DebugJSON("[DEBUG] Sending 2FA Response:", response)

	return response, nil
}

// #region Resend2FACode
func (s *authService) Resend2FACode(ctx context.Context, email string, token string) error {
	log := shared.GetLogger()

	// 1. Pobieramy istniejącą sesję 2FA
	session, err := s.cache.Get2FASession(ctx, token)
	if err != nil || session == nil {
		log.WarnMap("Resend2FA: session not found or expired", map[string]any{"token": token, "email": email})
		return errors.ErrSessionExpired
	}

	// 2. Weryfikacja zgodności adresu e-mail
	if !strings.EqualFold(session.Email, email) {
		log.WarnMap("SECURITY ALERT: Resend2FA email mismatch", map[string]any{
			"expected_email": session.Email,
			"provided_email": email,
		})
		return errors.ErrInvalidCredentials
	}

	// 3. Generujemy nowy bezpieczny kod OTP
	code, err := security.GenerateOTP(6)
	if err != nil {
		log.ErrorObj("Resend2FA: failed to generate OTP", err)
		return errors.ErrInternal
	}

	// 4. Hashujemy kod przed zapisem w pamięci podręcznej (przekazujemy string)
	codeBytes := []byte(code)
	defer clear(codeBytes)

	hashedCode, err := security.HashPassword(codeBytes, nil)
	if err != nil {
		log.ErrorObj("Resend2FA: failed to hash OTP", err)
		return errors.ErrInternal
	}

	// 5. Aktualizujemy podmieniony hash w sesji
	session.CodeHash = hashedCode

	// 6. Odświeżamy sesję 2FA w Redis (z resetem TTL na kolejne 5 minut)
	if err := s.cache.Set2FASession(ctx, token, *session, 5*time.Minute); err != nil {
		log.ErrorObj("Resend2FA: failed to update 2FA session in Redis", err)
		return errors.ErrInternal
	}

	log.DebugInfo("Resent 2FA code successfully", map[string]any{
		"email": email,
		"token": token,
		"code":  code,
	})

	return nil
}

// region AttemptLogin
// #region AttemptLogin
func (s *authService) AttemptLogin(ctx context.Context, email string, password []byte, fingerprint string) (*http.LoginResponse, error) {
	defer func() {
		if len(password) > 0 {
			for i := range password {
				password[i] = 0
			}
		}
	}()

	log := shared.GetLogger()

	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, errors.ErrInvalidCredentials
	}

	if err = s.CanUserLogin(user); err != nil {
		return nil, err
	}

	valid, err := security.VerifyPassword(password, user.Password, nil)
	if err != nil || !valid {
		return nil, s.handleFailedLogin(ctx, user.ID)
	}

	if user.FailedLoginAttempts > 0 {
		_ = s.userRepo.ResetFailedLoginAttempts(user.ID)
	}

	// 1. Odgałęzienie dla Urzędnika
	if user.Role != model.RoleCitizen && user.Role != model.RoleUser {
		log.Info("Procesowanie logowania urzędnika", map[string]any{"uid": user.ID, "role": user.Role})
		return s.prepareEmployeeLogin(ctx, user, fingerprint)
	}

	// 2. Odgałęzienie dla Zaufanego Urządzenia (Obywatel)
	device, err := s.userRepo.GetDeviceByFingerprint(ctx, user.ID, fingerprint)
	log.DebugDB("SCENARIUSZ A", device)

	if err == nil && device != nil && device.IsVerified && device.IsActive {
		return s.preparePreTrustSession(ctx, user, device.PublicKey, fingerprint)
	}

	// 3. Pozostałe scenariusze
	if user.TwoFactorEnabled {
		return s.prepare2FASession(ctx, user, fingerprint)
	}

	return s.finalizeLogin(ctx, user, fingerprint)
}

// region prepareEmployeeLogin
// #region prepareEmployeeLogin
func (s *authService) prepareEmployeeLogin(ctx context.Context, user *model.User, fingerprint string) (*http.LoginResponse, error) {
	log := shared.GetLogger()

	// 1. Pobieramy profil pracownika
	empProfile, err := s.employeeRepo.GetProfileByUserID(ctx, user.ID)
	if err != nil || !empProfile.Active {
		log.Error("Użytkownik ma rolę urzędniczą, ale brak aktywnego profilu", err)
		return nil, errors.ErrUnauthorized
	}

	// 2. Pobieramy aktywną kartę / poświadczenie fizyczne
	credential, err := s.employeeRepo.GetActiveCredentialByUserID(ctx, user.ID)
	if err != nil || credential.Status != model.EmployeeCredentialActive {
		log.WarnMap("Brak aktywnej karty urzędniczej dla użytkownika", map[string]any{
			"uid": user.ID,
		})
		return nil, errors.ErrInvalidCredentials
	}

	// 3. Budujemy bilet i sesję w Redis
	setupToken, sessionID, challenge, err := s.createChallengeSession(ctx, user.ID, fingerprint)
	if err != nil {
		return nil, err
	}

	// 4. Konstruujemy dane sesji z kontekstem urzędnika i kluczem z KARTY
	sessionData := redis.UserSession{
		UserID:      user.ID.String(),
		Fingerprint: shared.HashSHA256(fingerprint),
		PublicKey:   credential.PublicKey,
		Role:        string(user.Role),
	}

	if err := s.cache.SetSetupSession(ctx, sessionID, sessionData, 15*time.Minute); err != nil {
		log.ErrorObj("Failed to save employee setup session in Redis", err)
		return nil, errors.ErrInternal
	}

	return &http.LoginResponse{
		Type:       "employeeTrust",
		Challenge:  challenge,
		SetupToken: setupToken,
	}, nil
}

// region createChallengeSession
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

// region prepare2FASession
// #region prepare2FASession
func (s *authService) prepare2FASession(ctx context.Context, user *model.User, fingerprint string) (*http.LoginResponse, error) {
	log := shared.GetLogger()
	// 1. Generujemy 6-cyfrowy kod (bezpiecznie)
	code, err := security.GenerateOTP(6)
	if err != nil {
		return nil, errors.ErrInternal
	}
	codeBytes := []byte(code)
	defer clear(codeBytes)

	// 2. Hashujemy kod przed zapisem (Security: At-Rest protection)
	hashedCode, err := security.HashPassword(codeBytes, nil)
	if err != nil {
		return nil, errors.ErrInternal
	}

	// 3. Tworzymy sesję 2FA
	token := shared.GenerateSessionID()
	session := redis.TwoFASession{
		UserID:      user.ID.String(),
		Email:       user.Email,
		Token:       token,
		CodeHash:    hashedCode,
		Fingerprint: shared.HashSHA256(fingerprint),
		Attempts:    0,
	}

	// 4. Zapis do Redis (Metoda sama robi Marshal i dodaje prefix klucza)
	if err := s.cache.Set2FASession(ctx, token, session, 5*time.Minute); err != nil {
		log.ErrorObj("Failed to save 2FA session in Redis", err)
		return nil, errors.ErrInternal
	}

	// 5. TODO: Wyślij kod do użytkownika
	// s.emailService.Send2FACode(user.Email, code)

	// DEBUG
	log.DebugInfo("Generated 2FA code", map[string]any{
		"email": user.Email,
		"token": token,
		"code":  code,
	})

	return &http.LoginResponse{
		Type:          "2fa",
		TwoFARequired: true,
		TwoFAToken:    token,
	}, nil
}

// #region finalizeLogin
func (s *authService) finalizeLogin(ctx context.Context, user *model.User, fingerprint string) (*http.LoginResponse, error) {
	accessToken, sessionID, err := s.CreateAccessToken(ctx, user.ID, fingerprint)
	if err != nil {
		return nil, errors.ErrInternal
	}

	hashedFpt := shared.HashSHA256(fingerprint)

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

	sessionData := redis.UserSession{
		UserID:         user.ID.String(),
		Username:       user.Username,
		Email:          user.Email,
		Role:           string(user.Role),
		EmployeeNumber: empNumber,
		InstitutionID:  instID,
		DepartmentID:   deptID,
		Permissions:    permissions,
		Fingerprint:    hashedFpt,
	}

	if err := s.cache.SetSession(ctx, sessionID, sessionData, s.cfg.Session.TTL); err != nil {
		return nil, errors.ErrInternal
	}

	refreshToken, err := s.CreateRefreshToken(user.ID, fingerprint, nil)
	if err != nil {
		return nil, errors.ErrInternal
	}

	return &http.LoginResponse{
		TwoFARequired: false,
		AccessToken:   accessToken,
		RefreshToken:  refreshToken.Token,
		UserID:        user.ID.String(),
		ExpiresAt:     refreshToken.ExpiresAt.Unix(),
	}, nil
}

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

// region CreateAccessToken
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

// region CreateSetupToken
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

// region CreateRefreshToken
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

	if err := s.userRepo.CreateUser(u); err != nil {
		return nil, err
	}

	return u, nil
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

// region CanUserLogin
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

// region handleFailedLogin
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

// region preparePreTrustSession
// #region preparePreTrustSession
func (s *authService) preparePreTrustSession(ctx context.Context, user *model.User, publicKey string, fingerprint string) (*http.LoginResponse, error) {
	log := shared.GetLogger()

	// 1. Tworzymy bilet (SetupToken) i ID sesji
	setupToken, sessionID, challenge, err := s.createChallengeSession(ctx, user.ID, fingerprint)
	if err != nil {
		return nil, err
	}

	// 2. Dane sesji dla zwykłego urządzenia
	sessionData := redis.UserSession{
		UserID:      user.ID.String(),
		Fingerprint: shared.HashSHA256(fingerprint),
		PublicKey:   publicKey,
		Role:        string(user.Role),
	}

	// 3. Zapis w Redis
	if err := s.cache.SetSetupSession(ctx, sessionID, sessionData, 15*time.Minute); err != nil {
		log.ErrorObj("Failed to save setup session in Redis", err)
		return nil, errors.ErrInternal
	}

	log.DebugInfo("Pre-trust session prepared", map[string]any{
		"uid": user.ID,
		"sid": sessionID,
	})

	return &http.LoginResponse{
		Type:       "preTrust",
		Challenge:  challenge,
		SetupToken: setupToken,
		IsTrusted:  true,
	}, nil
}

// decodeChallenge próbuje zdekodować challenge zarówno w formacie StdBase64 jak i Base64URL
func decodeChallenge(storedChallenge string) ([]byte, error) {
	// 1. Zastąp znaki URL-safe na standardowy Base64
	b64 := strings.ReplaceAll(storedChallenge, "-", "+")
	b64 = strings.ReplaceAll(b64, "_", "/")

	// 2. Uzupełnij padding '=' jeśli brakuje
	switch len(b64) % 4 {
	case 2:
		b64 += "=="
	case 3:
		b64 += "="
	}

	// 3. Dekoduj standardowym dekoderem
	return base64.StdEncoding.DecodeString(b64)
}
