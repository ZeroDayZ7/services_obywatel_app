// cmdr: internal\service\reset_password_service.go

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/security"
	"github.com/zerodayz7/platform/services/auth-service/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type ResetSession struct {
	UserID          string `json:"user_id"`
	Email           string `json:"email"`
	AgreementNumber string `json:"agreement_number"`
	CodeHash        string `json:"code"`
	Token           string `json:"token"`
	Challenge       string `json:"challenge"`
	Attempts        int    `json:"attempts"`
	Verified        bool   `json:"verified"`
}

type DeviceInfo struct {
	Fingerprint string
	PublicKey   string
	DeviceName  string
	Platform    string
	IP          string
}

// region interface
type PasswordResetService interface {
	StartResetProcess(ctx context.Context, agreementNumber string, value string, method string) (string, error)
	VerifyCode(ctx context.Context, token, code string) (*ResetSession, error)
	FinalizeReset(ctx context.Context, token, newPassword string) error
}

type passwordResetService struct {
	userRepo         repository.UserRepository
	refreshTokenRepo repository.RefreshTokenRepository
	cache            *redis.Cache
}

//#region NewPasswordResetService
func NewPasswordResetService(
	userRepo repository.UserRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	cache *redis.Cache,
) PasswordResetService {
	return &passwordResetService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		cache:            cache,
	}
}

// region StartResetProcess
//#region StartResetProcess
func (s *passwordResetService) StartResetProcess(ctx context.Context, agreementNumber string, value string, method string) (string, error) {
	user, err := s.userRepo.GetUserByEmail(ctx, value)
	if err != nil {
		return "", errors.ErrEmailIsSendIfExists
	}

	// Token sesji resetu jako cryptographically secure random string (nie UUID v7)
	token, err := security.GenerateRandomString(32)
	if err != nil {
		return "", errors.ErrInternal
	}

	code, err := security.GenerateOTPString(6)
	if err != nil {
		return "", errors.ErrInternal
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return "", errors.ErrInternal
	}

	session := ResetSession{
		UserID:          user.ID.String(),
		Email:           user.Email,
		AgreementNumber: agreementNumber,
		CodeHash:        string(hashed),
		Token:           token,
		Attempts:        0,
	}

	if err := s.saveSession(ctx, token, &session); err != nil {
		return "", errors.ErrInternal
	}

	fmt.Printf("[RESET DEBUG] Kod dla umowy %s (%s): %s\n", agreementNumber, value, code)
	return token, nil
}

//#region VerifyCode
func (s *passwordResetService) VerifyCode(ctx context.Context, token, code string) (*ResetSession, error) {
	session, err := s.getSession(ctx, token)
	if err != nil {
		return nil, errors.ErrResetSessionNotFound
	}

	if session.Attempts >= 5 {
		return nil, errors.Err2FALocked
	}

	if err := bcrypt.CompareHashAndPassword([]byte(session.CodeHash), []byte(code)); err != nil {
		session.Attempts++
		_ = s.saveSession(ctx, token, session)
		return nil, errors.ErrInvalidResetCode
	}

	// Challenge po zweryfikowaniu kodu OTP również jako bezpieczny losowy string
	challenge, err := security.GenerateRandomString(32)
	if err != nil {
		return nil, errors.ErrInternal
	}

	session.Verified = true
	session.Challenge = challenge
	if err := s.saveSession(ctx, token, session); err != nil {
		return nil, errors.ErrInternal
	}

	return session, nil
}

// region FinalizeReset
//#region FinalizeReset
func (s *passwordResetService) FinalizeReset(ctx context.Context, token, newPassword string) error {
	session, err := s.getSession(ctx, token)
	if err != nil {
		return errors.ErrResetSessionNotFound
	}

	if !session.Verified {
		return errors.ErrUnauthorized
	}

	userUUID, err := uuid.Parse(session.UserID)
	if err != nil {
		return errors.ErrInvalidRequest
	}

	user, err := s.userRepo.GetByID(ctx, userUUID)
	if err != nil {
		return errors.ErrUserNotFound
	}

	// 1. Hashowanie hasła za pomocą Argon2id
	passBytes := []byte(newPassword)
	defer clear(passBytes)

	hashedPassword, err := security.HashPassword(passBytes, nil)
	if err != nil {
		return errors.ErrInternal
	}

	// 2. Aktualizacja danych użytkownika
	now := time.Now()
	user.Password = hashedPassword
	user.PasswordChangedAt = &now
	user.FailedLoginAttempts = 0
	user.LockedUntil = nil

	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	// 3. Unieważnienie wszystkich aktywnych sesji/tokenów użytkownika
	_ = s.refreshTokenRepo.RevokeAllUserTokens(ctx, userUUID)

	// 4. Czyszczenie sesji resetu w Redis
	_ = s.cache.Del(ctx, fmt.Sprintf("reset:password:%s", token))

	return nil
}

// region getSession
//#region getSession
func (s *passwordResetService) getSession(ctx context.Context, token string) (*ResetSession, error) {
	key := fmt.Sprintf("reset:password:%s", token)
	data, err := s.cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	var session ResetSession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, err
	}
	return &session, nil
}

//#region saveSession
func (s *passwordResetService) saveSession(ctx context.Context, token string, session *ResetSession) error {
	key := fmt.Sprintf("reset:password:%s", token)
	data, _ := json.Marshal(session)
	return s.cache.Set(ctx, key, data, 10*time.Minute)
}
