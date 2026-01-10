package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/viper"
	"github.com/zerodayz7/platform/services/auth-service/internal/features/auth/model"
	authModel "github.com/zerodayz7/platform/services/auth-service/internal/features/auth/model"
	authRepo "github.com/zerodayz7/platform/services/auth-service/internal/features/auth/repository"
	userRepo "github.com/zerodayz7/platform/services/auth-service/internal/features/users/repository"

	"github.com/golang-jwt/jwt/v5"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/auth-service/internal/shared/security"
)

type AuthService struct {
	repo        userRepo.UserRepository
	refreshRepo authRepo.RefreshTokenRepository
	cfg         *viper.Config
}

func NewAuthService(
	repo userRepo.UserRepository,
	refreshRepo authRepo.RefreshTokenRepository,
	cfg *viper.Config,
) *AuthService {
	return &AuthService{
		repo:        repo,
		refreshRepo: refreshRepo,
		cfg:         cfg,
	}
}

// Pobranie użytkownika po ID
func (s *AuthService) UpdatePassword(
	ctx context.Context,
	userID uuid.UUID,
	newPassword string,
) error {
	log := shared.GetLogger()

	// Pobranie użytkownika po ID
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		log.ErrorMap("GetByID failed", map[string]any{"userID": userID, "error": err.Error()})
		return err
	}
	if user == nil {
		return errors.ErrUserNotFound
	}

	// Hashowanie nowego hasła
	hashed, err := security.HashPassword(newPassword)
	if err != nil {
		log.ErrorMap("HashPassword failed", map[string]any{"error": err.Error()})
		return err
	}

	// Aktualizacja hasła w bazie
	user.Password = hashed
	if err := s.repo.Update(ctx, user); err != nil {
		log.ErrorMap("Update user failed", map[string]any{"userID": userID, "error": err.Error()})
		return err
	}

	log.InfoMap("Password updated successfully", map[string]any{"userID": userID})
	return nil
}

// CreateAccessToken generuje access token i zwraca sessionID
// Dodajemy fingerprint jako argument, aby "przypiąć" token do urządzenia
func (s *AuthService) CreateAccessToken(userID uuid.UUID, fingerprint string) (accessToken string, sessionID string, err error) {
	sessionID = shared.GenerateUuid()

	claims := jwt.MapClaims{
		"uid": userID,
		"sid": sessionID,
		"fpt": fingerprint,
	}

	accessToken, err = security.GenerateJWT(claims, s.cfg.JWT.AccessSecret, s.cfg.JWT.AccessTTL)
	if err != nil {
		return "", "", err
	}

	// Zwracamy accessToken i sessionID zgodnie z Twoim oryginalnym zamysłem
	return accessToken, sessionID, nil
}

// Opcjonalnie – wygodne metody dla Redis (możesz używać w handlerze)
func (s *AuthService) CreateSession(userID uuid.UUID) (string, error) {
	sessionID := shared.GenerateUuid()
	return sessionID, nil
}

func (s *AuthService) CreateRefreshToken(userID uuid.UUID, fingerprint string) (*authModel.RefreshToken, error) {
	refreshTTL := s.cfg.JWT.RefreshTTL

	// 1. Generujemy surowy, losowy token (64 znaki)
	rawToken, err := security.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	// 2. Hashujemy surowy token za pomocą SHA-256
	// Robimy to, aby w bazie nie było widać "hasła" sesji
	hash := sha256.Sum256([]byte(rawToken))
	hashedTokenHex := hex.EncodeToString(hash[:])

	// 3. Tworzymy obiekt do zapisu w DB (z zahashowanym tokenem)
	rt := &authModel.RefreshToken{
		UserID:            userID,
		Token:             hashedTokenHex,
		DeviceFingerprint: fingerprint,
		Revoked:           false,
		CreatedAt:         time.Now(),
		ExpiresAt:         time.Now().Add(refreshTTL),
	}

	// 4. Zapisujemy w bazie danych
	if err := s.refreshRepo.Save(rt); err != nil {
		return nil, err
	}

	// 5. Nadpisujemy pole Token surową wartością PRZED zwróceniem do Handlera
	// Dzięki temu Handler wyśle do Fluttera "SECRET_123", a nie "HASH_456"
	rt.Token = rawToken

	return rt, nil
}

func (s *AuthService) UpdateRefreshTokensFingerprint(userID uuid.UUID, oldFP, newFP string) error {
	// Delegacja do repozytorium, które już masz zaimplementowane
	return s.refreshRepo.UpdateFingerprintByUser(userID, oldFP, newFP)
}

func (s *AuthService) GetRefreshToken(token string) (*authModel.RefreshToken, error) {
	return s.refreshRepo.GetByToken(token)
}

func (s *AuthService) RevokeRefreshToken(token string) error {
	rt, err := s.refreshRepo.GetByToken(token)
	if err != nil {
		return err
	}
	rt.Revoked = true
	return s.refreshRepo.Update(rt)
}

func (s *AuthService) IsEmailExists(email string) (bool, error) {
	log := shared.GetLogger()
	log.DebugMap("IsEmailExists", map[string]any{"email": email})

	exists, err := s.repo.EmailExists(email)
	if err != nil {
		log.ErrorMap("repo.EmailExists failed", map[string]any{"error": err.Error()})
		return false, err
	}
	return exists, nil
}

func (s *AuthService) IsUsernameExists(username string) (bool, error) {
	log := shared.GetLogger()
	log.DebugMap("IsUsernameExists", map[string]any{"username": username})

	exists, err := s.repo.UsernameExists(username)
	if err != nil {
		log.ErrorMap("repo.UsernameExists failed", map[string]any{"error": err.Error()})
		return false, err
	}
	return exists, nil
}

func (s *AuthService) IsEmailOrUsernameExists(email, username string) (bool, bool, error) {
	existsEmail, existsUsername, err := s.repo.EmailOrUsernameExists(email, username)
	if err != nil {
		log := shared.GetLogger()
		log.ErrorMap("repo.EmailOrUsernameExists failed", map[string]any{"error": err.Error()})
		return false, false, err
	}
	return existsEmail, existsUsername, nil
}

func (s *AuthService) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	u, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errors.ErrUserNotFound
	}
	return u, nil
}

func (s *AuthService) GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errors.ErrUserNotFound
	}
	return u, nil
}

// W auth_service.go
func (s *AuthService) VerifyPassword(user *model.User, password []byte) (bool, error) {
	return security.VerifyPassword(password, user.Password)
}

func (s *AuthService) Register(username, email, rawPassword string) (*model.User, error) {
	log := shared.GetLogger()
	log.DebugMap("Register attempt", map[string]any{"email": email, "username": username})

	emailExists, usernameExists, err := s.repo.EmailOrUsernameExists(email, username)
	if err != nil {
		log.ErrorMap("EmailOrUsernameExists failed", map[string]any{"error": err.Error()})
		return nil, fmt.Errorf("checking email/username existence: %w", err)
	}

	if emailExists {
		log.WarnMap("Email already registered", map[string]any{"email": email})
		return nil, errors.ErrEmailExists
	}
	if usernameExists {
		log.WarnMap("Username already exists", map[string]any{"username": username})
		return nil, errors.ErrUsernameExists
	}

	hash, err := security.HashPassword(rawPassword)
	if err != nil {
		log.ErrorMap("Password hashing failed", map[string]any{"error": err.Error()})
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	u := &model.User{
		Username: username,
		Email:    email,
		Password: hash,
	}

	if err := s.repo.CreateUser(u); err != nil {
		log.ErrorMap("CreateUser failed", map[string]any{"error": err.Error()})
		return nil, fmt.Errorf("creating user: %w", err)
	}

	log.InfoMap("User registered successfully", map[string]any{"email": email, "username": username})
	return u, nil
}

// RegisterUserDevice przyjmuje czyste dane, nie wie nic o pakiecie validator
func (s *AuthService) RegisterUserDevice(
	ctx context.Context,
	userID uuid.UUID,
	fingerprint string,
	publicKey string,
	deviceName string,
	platform string,
	isVerified bool,
	lastIp string,
) error {
	log := shared.GetLogger()

	device := authModel.UserDevice{
		UserID:              userID,
		DeviceFingerprint:   fingerprint,
		PublicKey:           publicKey,
		DeviceNameEncrypted: deviceName,
		Platform:            platform,
		IsVerified:          isVerified,
		LastIp:              lastIp,
		IsActive:            true,
	}

	err := s.repo.SaveDevice(ctx, &device)
	if err != nil {
		log.ErrorObj("Failed to save device", err)
		return errors.ErrInternal
	}

	return nil
}

func (s *AuthService) IncrementUserFailedLogin(userID uuid.UUID) error {
	return s.repo.IncrementFailedLogin(userID)
}

func (s *AuthService) ResetUserFailedLogin(userID uuid.UUID) error {
	return s.repo.ResetFailedLogin(userID)
}

func (s *AuthService) RepoUpdateUser(ctx context.Context, user *model.User) error {
	return s.repo.Update(ctx, user)
}

func (s *AuthService) CanUserLogin(user *model.User) *errors.AppError {
	switch user.Status {
	case model.StatusSuspended:
		return errors.ErrAccountSuspended
	case model.StatusBanned:
		return errors.ErrAccountBanned
	case model.StatusPending:
		return errors.ErrAccountPending
	case model.StatusActive:
		return nil
	default:
		return errors.ErrInternal
	}
}

func (s *AuthService) GetDeviceByFingerprint(ctx context.Context, userID uuid.UUID, fingerprint string) (*authModel.UserDevice, error) {
	return s.repo.GetDeviceByFingerprint(ctx, userID, fingerprint)
}

func (s *AuthService) ActivateDevice(ctx context.Context, deviceID uuid.UUID, publicKey string, deviceName string) error {
	return s.repo.UpdateDeviceStatus(ctx, deviceID, publicKey, deviceName, true, true)
}
