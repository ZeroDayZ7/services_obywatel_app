package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/redis"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/auth-service/internal/model"
	"github.com/zerodayz7/platform/services/auth-service/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type ResetSession struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	CodeHash  string `json:"code"`
	Token     string `json:"token"`
	Challenge string `json:"challenge"`
	Attempts  int    `json:"attempts"`
	Verified  bool   `json:"verified"`
}

type DeviceInfo struct {
	Fingerprint string
	PublicKey   string
	DeviceName  string
	Platform    string
	IP          string
}

type PasswordResetService interface {
	StartResetProcess(ctx context.Context, email string) (string, error)
	VerifyCode(ctx context.Context, token, code string) (*ResetSession, error)
	FinalizeReset(ctx context.Context, token, newPassword, signature string, device DeviceInfo) error
}

type resetRepository interface {
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	GetDeviceByFingerprint(ctx context.Context, userID uuid.UUID, fingerprint string) (*model.UserDevice, error)
	SaveDevice(ctx context.Context, device *model.UserDevice) error
}

type passwordResetService struct {
	userRepo         resetRepository
	refreshTokenRepo repository.RefreshTokenRepository
	cache            *redis.Cache
}

func NewPasswordResetService(
	userRepo repository.UserRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	cache *redis.Cache, // 3. Cache
) PasswordResetService {
	return &passwordResetService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		cache:            cache,
	}
}

func (s *passwordResetService) StartResetProcess(ctx context.Context, email string) (string, error) {
	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return "", errors.ErrEmailIsSendIfExists
	}

	token := shared.GenerateUuidV7()
	code := fmt.Sprintf("%06d", shared.RandInt(100000, 999999))
	hashed, _ := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)

	session := ResetSession{
		UserID:   user.ID.String(),
		Email:    user.Email,
		CodeHash: string(hashed),
		Token:    token,
		Attempts: 0,
	}

	if err := s.saveSession(ctx, token, &session); err != nil {
		return "", errors.ErrInternal
	}

	fmt.Printf("[RESET DEBUG] Kod dla %s: %s\n", email, code)
	return token, nil
}

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

	session.Verified = true
	session.Challenge = shared.GenerateUuidV7()
	if err := s.saveSession(ctx, token, session); err != nil {
		return nil, errors.ErrInternal
	}

	return session, nil
}

func (s *passwordResetService) FinalizeReset(ctx context.Context, token, newPassword, signature string, device DeviceInfo) error {
	session, err := s.getSession(ctx, token)
	if err != nil {
		return errors.ErrResetSessionNotFound
	}

	if !session.Verified {
		return errors.ErrUnauthorized
	}

	userUUID, _ := uuid.Parse(session.UserID)

	var pubKeyToVerify string
	existingDevice, err := s.userRepo.GetDeviceByFingerprint(ctx, userUUID, device.Fingerprint)

	if err != nil {
		if device.PublicKey == "" {
			return errors.ErrUntrustedDevice
		}
		newDevice := &model.UserDevice{
			ID:     uuid.New(),
			UserID: userUUID,
			// Fingerprint: device.Fingerprint,
			PublicKey: device.PublicKey,
			// DeviceName:  device.DeviceName,
			Platform: device.Platform,
			// LastIP:      device.IP,
			IsVerified: false,
			IsActive:   true,
		}
		if err := s.userRepo.SaveDevice(ctx, newDevice); err != nil {
			return errors.ErrInternal
		}
		pubKeyToVerify = device.PublicKey
	} else {
		pubKeyToVerify = existingDevice.PublicKey
	}

	challenge := fmt.Sprintf("%s|%s", session.Challenge, token)
	if !shared.VerifyEd25519Signature(pubKeyToVerify, challenge, signature) {
		return errors.ErrVerificationFailed
	}

	user, err := s.userRepo.GetByID(ctx, userUUID)
	if err != nil {
		return errors.ErrUserNotFound
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	user.Password = string(hashedPassword)

	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	_ = s.refreshTokenRepo.RevokeAllUserTokens(ctx, userUUID)

	_ = s.cache.Del(ctx, fmt.Sprintf("reset:password:%s", token))
	return nil
}

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

func (s *passwordResetService) saveSession(ctx context.Context, token string, session *ResetSession) error {
	key := fmt.Sprintf("reset:password:%s", token)
	data, _ := json.Marshal(session)
	return s.cache.Set(ctx, key, data, 10*time.Minute)
}
