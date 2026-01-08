package db

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/auth-service/internal/features/auth/model"
	authModel "github.com/zerodayz7/platform/services/auth-service/internal/features/auth/model"
	"github.com/zerodayz7/platform/services/auth-service/internal/features/users/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var _ repository.UserRepository = (*UserRepo)(nil)

type UserRepo struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) CreateUser(user *model.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepo) GetByID(id uuid.UUID) (*model.User, error) {
	var u model.User
	if err := r.db.First(&u, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) GetByEmail(email string) (*model.User, error) {
	var u model.User
	if err := r.db.Where("email = ?", email).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) EmailExists(email string) (bool, error) {
	var count int64
	if err := r.db.Model(&model.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *UserRepo) UsernameExists(username string) (bool, error) {
	var count int64
	if err := r.db.Model(&model.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *UserRepo) EmailOrUsernameExists(email, username string) (emailExists, usernameExists bool, err error) {
	var users []model.User
	if err := r.db.Where("email = ? OR username = ?", email, username).Find(&users).Error; err != nil {
		return false, false, err
	}

	for _, u := range users {
		if u.Email == email {
			emailExists = true
		}
		if u.Username == username {
			usernameExists = true
		}
	}
	return
}

func (r *UserRepo) Update(user *model.User) error {
	return r.db.Save(user).Error
}

func (r *UserRepo) SaveDevice(ctx context.Context, device *model.UserDevice) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		// Klucz unikalny to kombinacja UserID i Fingerprint
		Columns: []clause.Column{{Name: "user_id"}, {Name: "device_fingerprint"}},
		// Co ma się stać przy konflikcie? Aktualizujemy dane:
		DoUpdates: clause.AssignmentColumns([]string{
			"public_key",
			"device_name_encrypted",
			"platform",
			"is_active",
			"last_used_at",
		}),
	}).Create(device).Error
}

func (r *UserRepo) UpdateFailedLogin(userID uuid.UUID, attempts int) error {
	return r.db.Model(&model.User{}).
		Where("id = ?", userID).
		Update("failed_login_attempts", attempts).Error
}

func (r *UserRepo) GetDeviceByFingerprint(ctx context.Context, userID uuid.UUID, fingerprint string) (*authModel.UserDevice, error) {
	var device authModel.UserDevice
	// Poprawka: używamy r.db (małe litery, zgodnie z definicją struktury)
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND device_fingerprint = ? AND is_active = ?", userID, fingerprint, true).
		First(&device).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

func (r *UserRepo) UpdateDeviceStatus(ctx context.Context, deviceID uuid.UUID, publicKey string, deviceName string, isActive bool, isVerified bool) error {
	return r.db.WithContext(ctx).Model(&authModel.UserDevice{}).
		Where("id = ?", deviceID).
		Updates(map[string]interface{}{
			"public_key":            publicKey,
			"device_name_encrypted": deviceName,
			"is_active":             isActive,
			"is_verified":           isVerified,
			"last_used_at":          time.Now(),
		}).Error
}
