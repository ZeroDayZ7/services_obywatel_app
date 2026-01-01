package mysql

import (
	"time"

	"github.com/zerodayz7/platform/services/auth-service/internal/features/auth/model"
	"gorm.io/gorm"
)

type RefreshTokenRepository struct {
	DB *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{DB: db}
}

// Save — dodajemy parametr fingerprint
func (r *RefreshTokenRepository) Save(userID uint, token string, fingerprint string, duration time.Duration) error {
	rt := model.RefreshToken{
		UserID:            userID,
		Token:             token,
		DeviceFingerprint: fingerprint,
		ExpiresAt:         time.Now().Add(duration),
		Revoked:           false,
	}
	return r.DB.Create(&rt).Error
}

func (r *RefreshTokenRepository) Get(token string) (*model.RefreshToken, error) {
	var rt model.RefreshToken
	err := r.DB.Where("token = ? AND revoked = false AND expires_at > ?", token, time.Now()).First(&rt).Error
	if err != nil {
		return nil, err
	}
	return &rt, nil
}

func (r *RefreshTokenRepository) Revoke(token string) error {
	return r.DB.Model(&model.RefreshToken{}).Where("token = ?", token).Update("revoked", true).Error
}

func (r *RefreshTokenRepository) GetByToken(token string) (*model.RefreshToken, error) {
	var rt model.RefreshToken
	err := r.DB.Where("token = ?", token).First(&rt).Error
	if err != nil {
		return nil, err
	}
	return &rt, nil
}

func (r *RefreshTokenRepository) Update(rt *model.RefreshToken) error {
	return r.DB.Save(rt).Error
}
