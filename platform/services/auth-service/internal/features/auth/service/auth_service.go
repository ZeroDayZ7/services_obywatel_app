package service

import (
	"fmt"
	"time"

	"github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/services/auth-service/config"
	authModel "github.com/zerodayz7/platform/services/auth-service/internal/features/auth/model"
	authRepo "github.com/zerodayz7/platform/services/auth-service/internal/features/auth/repository"
	"github.com/zerodayz7/platform/services/auth-service/internal/features/users/model"
	userRepo "github.com/zerodayz7/platform/services/auth-service/internal/features/users/repository"

	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/auth-service/internal/shared/security"
)

type AuthService struct {
	repo        userRepo.UserRepository
	refreshRepo authRepo.RefreshTokenRepository
}

func NewAuthService(repo userRepo.UserRepository, refreshRepo authRepo.RefreshTokenRepository) *AuthService {
	return &AuthService{
		repo:        repo,
		refreshRepo: refreshRepo,
	}
}

func (s *AuthService) CreateRefreshToken(userID uint) (*authModel.RefreshToken, error) {
	refreshTTL := config.AppConfig.JWT.RefreshTTL

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

	if err := s.refreshRepo.Save(userID, token, refreshTTL); err != nil {
		return nil, err
	}

	return rt, nil
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
	return security.VerifyPassword(password, user.Password)
}

func (s *AuthService) Verify2FACodeByID(userID uint, code string) (bool, error) {
	log := shared.GetLogger()
	log.DebugMap("Verify2FACodeByID", map[string]any{"userID": userID})

	u, err := s.repo.GetByID(userID)
	if err != nil {
		log.ErrorMap("GetByID failed", map[string]any{"error": err.Error()})
		return false, err
	}
	if !u.TwoFactorEnabled {
		log.WarnMap("2FA not enabled", map[string]any{"userID": userID})
		return false, nil
	}

	return code == u.TwoFactorSecret, nil
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
