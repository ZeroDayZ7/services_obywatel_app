package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/errors"
	"github.com/zerodayz7/platform/pkg/security"
	"github.com/zerodayz7/platform/services/auth-service/internal/model"
)

type changePasswordUserRepo struct {
	user *model.User
}

func (r *changePasswordUserRepo) CreateUser(*model.User) error { return nil }
func (r *changePasswordUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return r.user, nil
}
func (r *changePasswordUserRepo) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	return nil, nil
}
func (r *changePasswordUserRepo) DeleteDevice(ctx context.Context, userID uuid.UUID, fingerprint string) error {
	return nil
}
func (r *changePasswordUserRepo) EmailExists(string) (bool, error)    { return false, nil }
func (r *changePasswordUserRepo) UsernameExists(string) (bool, error) { return false, nil }
func (r *changePasswordUserRepo) EmailOrUsernameExists(email, username string) (bool, bool, error) {
	return false, false, nil
}
func (r *changePasswordUserRepo) Update(ctx context.Context, user *model.User) error {
	r.user = user
	return nil
}
func (r *changePasswordUserRepo) SaveDevice(ctx context.Context, device *model.UserDevice) error {
	return nil
}
func (r *changePasswordUserRepo) IncrementUserFailedLogin(ctx context.Context, userID uuid.UUID) (int8, error) {
	return 0, nil
}
func (r *changePasswordUserRepo) LockUserTemporarily(userID uuid.UUID, duration time.Duration) error {
	return nil
}
func (r *changePasswordUserRepo) ResetFailedLoginAttempts(userID uuid.UUID) error { return nil }
func (r *changePasswordUserRepo) PermanentLock(ctx context.Context, userID uuid.UUID) error {
	return nil
}
func (r *changePasswordUserRepo) GetDeviceByFingerprint(ctx context.Context, userID uuid.UUID, fingerprint string) (*model.UserDevice, error) {
	return nil, nil
}

type changePasswordRefreshRepo struct{}

func (r *changePasswordRefreshRepo) Save(*model.RefreshToken) error                 { return nil }
func (r *changePasswordRefreshRepo) Get(string) (*model.RefreshToken, error)        { return nil, nil }
func (r *changePasswordRefreshRepo) Revoke(string) error                            { return nil }
func (r *changePasswordRefreshRepo) GetByToken(string) (*model.RefreshToken, error) { return nil, nil }
func (r *changePasswordRefreshRepo) Update(*model.RefreshToken) error               { return nil }
func (r *changePasswordRefreshRepo) RevokeByFingerprint(ctx context.Context, userID uuid.UUID, fingerprint string) error {
	return nil
}
func (r *changePasswordRefreshRepo) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	return nil
}
func (r *changePasswordRefreshRepo) GetSessions(ctx context.Context, userID uuid.UUID) ([]model.UserSessionDTO, error) {
	return nil, nil
}
func (r *changePasswordRefreshRepo) RevokeSession(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) error {
	return nil
}

func TestChangePasswordRequiresValidCurrentPasswordAndRevokesSessions(t *testing.T) {
	userID := uuid.New()
	currentHash, err := security.HashPassword([]byte("OldPassword1!"), nil)
	if err != nil {
		t.Fatal(err)
	}

	repo := &changePasswordUserRepo{user: &model.User{
		ID:       userID,
		Email:    "user@example.com",
		Password: currentHash,
	}}

	service := &authService{userRepo: repo, refreshRepo: &changePasswordRefreshRepo{}}

	if err := service.ChangePassword(context.Background(), userID, "WrongPassword1!", "NewPassword1!"); err == nil || err != errors.ErrInvalidCredentials {
		t.Fatalf("expected invalid credentials error, got %v", err)
	}

	if err := service.ChangePassword(context.Background(), userID, "OldPassword1!", "OldPassword1!"); err == nil {
		t.Fatal("expected request validation error for reusing the same password")
	}

	if err := service.ChangePassword(context.Background(), userID, "OldPassword1!", "NewPassword1!"); err != nil {
		t.Fatalf("expected password change to succeed, got %v", err)
	}

	if repo.user.Password == currentHash {
		t.Fatal("expected password hash to be updated")
	}
}
