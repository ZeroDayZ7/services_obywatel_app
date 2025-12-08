package service

import (
	"fmt"
	"time"

	"github.com/zerodayz7/http-server/config"
	"github.com/zerodayz7/http-server/internal/errors"
	authModel "github.com/zerodayz7/http-server/internal/features/auth/model"
	authRepo "github.com/zerodayz7/http-server/internal/features/auth/repository"
	"github.com/zerodayz7/http-server/internal/features/users/model"
	userRepo "github.com/zerodayz7/http-server/internal/features/users/repository"

	"github.com/zerodayz7/http-server/internal/shared/logger"
	"github.com/zerodayz7/http-server/internal/shared/security"

	"go.uber.org/zap"
)

type AuthService struct {
	repo        userRepo.UserRepository
	refreshRepo authRepo.RefreshTokenRepository
}

func NewAuthService(
	repo userRepo.UserRepository, refreshRepo authRepo.RefreshTokenRepository) *AuthService {
	return &AuthService{
		repo:        repo,
		refreshRepo: refreshRepo,
	}
}

// Tworzy nowy refresh token i zapisuje w DB
func (s *AuthService) CreateRefreshToken(userID uint) (*authModel.RefreshToken, error) {
	refreshTTL := config.AppConfig.JWT.RefreshTTL

	// Generowanie losowego refresh tokena
	token, err := security.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	rt := &authModel.RefreshToken{
		UserID:    userID,
		Token:     token,
		Revoked:   false,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(refreshTTL),
	}

	// Zapis do DB
	if err := s.refreshRepo.Save(userID, token, refreshTTL); err != nil {
		return nil, err
	}

	return rt, nil
}

// Pobiera refresh token z DB
func (s *AuthService) GetRefreshToken(token string) (*authModel.RefreshToken, error) {
	rt, err := s.refreshRepo.GetByToken(token)
	if err != nil {
		return nil, err
	}
	return rt, nil
}

// Unieważnia refresh token
func (s *AuthService) RevokeRefreshToken(token string) error {
	rt, err := s.refreshRepo.GetByToken(token)
	if err != nil {
		return err
	}
	rt.Revoked = true
	return s.refreshRepo.Update(rt)
}

func (s *AuthService) IsEmailExists(email string) (bool, error) {
	log := logger.GetLogger()
	log.Debug("IsEmailExists", zap.String("email", email))

	exists, err := s.repo.EmailExists(email)
	if err != nil {
		log.Error("repo.EmailExists failed", zap.Error(err))
		return false, err
	}
	return exists, nil
}

func (s *AuthService) IsUsernameExists(username string) (bool, error) {
	log := logger.GetLogger()
	log.Debug("IsUsernameExists", zap.String("username", username))

	exists, err := s.repo.UsernameExists(username)
	if err != nil {
		log.Error("repo.UsernameExists failed", zap.Error(err))
		return false, err
	}
	return exists, nil
}

func (s *AuthService) IsEmailOrUsernameExists(email, username string) (bool, bool, error) {
	existsEmail, existsUsername, err := s.repo.EmailOrUsernameExists(email, username)
	if err != nil {
		log := logger.GetLogger()
		log.Error("repo.EmailOrUsernameExists failed", zap.Error(err))
		return false, false, err
	}
	return existsEmail, existsUsername, nil
}

func (s *AuthService) GetUserByEmail(email string) (*model.User, error) {
	u, err := s.repo.GetByEmail(email)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errors.ErrUserNotFound
	}
	return u, nil
}

func (s *AuthService) VerifyPassword(user *model.User, password string) (bool, error) {
	valid, err := security.VerifyPassword(password, user.Password)
	if err != nil {
		return false, err
	}
	return valid, nil
}

func (s *AuthService) Verify2FACodeByID(userID uint, code string) (bool, error) {
	log := logger.GetLogger()
	log.Debug("Verify2FACodeByID", zap.Uint("userID", userID))

	u, err := s.repo.GetByID(userID)
	if err != nil {
		log.Error("GetByID failed", zap.Error(err))
		return false, err
	}
	if !u.TwoFactorEnabled {
		log.Warn("2FA not enabled", zap.Uint("userID", userID))
		return false, nil
	}

	return code == u.TwoFactorSecret, nil
}

func (s *AuthService) Register(username, email, rawPassword string) (*model.User, error) {
	log := logger.GetLogger()
	log.Debug("Register attempt", zap.String("email", email), zap.String("username", username))

	emailExists, usernameExists, err := s.repo.EmailOrUsernameExists(email, username)
	if err != nil {
		log.Error("EmailOrUsernameExists failed", zap.Error(err))
		return nil, fmt.Errorf("checking email/username existence: %w", err)
	}

	if emailExists {
		log.Warn("Email already registered", zap.String("email", email))
		return nil, errors.ErrEmailExists
	}
	if usernameExists {
		log.Warn("Username already exists", zap.String("username", username))
		return nil, errors.ErrUsernameExists
	}

	hash, err := security.HashPassword(rawPassword)
	if err != nil {
		log.Error("Password hashing failed", zap.Error(err))
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	u := &model.User{
		Username: username,
		Email:    email,
		Password: hash,
	}

	if err := s.repo.CreateUser(u); err != nil {
		log.Error("CreateUser failed", zap.Error(err))
		return nil, fmt.Errorf("creating user: %w", err)
	}

	log.Info("User registered successfully", zap.String("email", email), zap.String("username", username))
	return u, nil
}

// func (s *UserService) VerifyTwoFactorCode(userID, code string) (bool, error) {
// 	user, err := mysql.GetUserByID(userID)
// 	if err != nil {
// 		return false, err
// 	}

// 	if !user.TwoFactorEnabled {
// 		return false, errors.New("2FA not enabled for this user")
// 	}

// 	// Weryfikacja TOTP (np. Google Authenticator)
// 	valid := totp.Validate(code, user.TwoFactorSecret)
// 	if !valid {
// 		return false, nil
// 	}

// 	// Tutaj możesz np. zarejestrować timestamp ostatniego logowania 2FA
// 	user.Last2FALogin = time.Now()
// 	mysql.UpdateUser(user)

// 	return true, nil
// }
