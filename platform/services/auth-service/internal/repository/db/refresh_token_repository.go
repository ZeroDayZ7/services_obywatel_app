// platform\services\auth-service\internal\repository\db\refresh_token_repository.go
package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/services/auth-service/internal/model"
	repository "github.com/zerodayz7/platform/services/auth-service/internal/repository"
	"gorm.io/gorm"
)

var _ repository.RefreshTokenRepository = (*RefreshTokenRepository)(nil)

type RefreshTokenRepository struct {
	DB *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{DB: db}
}

// Save — dodaje nowy token z przypisanym fingerprintem urządzenia
func (r *RefreshTokenRepository) Save(rt *model.RefreshToken) error {
	return r.DB.Create(rt).Error
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

func (r *RefreshTokenRepository) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	return r.DB.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("user_id = ? AND revoked = ?", userID, false).
		Update("revoked", true).Error
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

func (r *RefreshTokenRepository) GetSessions(ctx context.Context, userID uuid.UUID) ([]model.UserSessionDTO, error) {
	var results []model.UserSessionDTO

	err := r.DB.WithContext(ctx).
		Table("refresh_tokens").
		Select(`
      refresh_tokens.id as session_id, 
      user_devices.device_name_encrypted, 
      user_devices.platform, 
      refresh_tokens.created_at, 
      user_devices.last_used_at, 
      refresh_tokens.device_fingerprint as fingerprint
    `).
		Joins("JOIN user_devices ON user_devices.device_fingerprint = refresh_tokens.device_fingerprint AND user_devices.user_id = refresh_tokens.user_id").
		Where("refresh_tokens.user_id = ? AND refresh_tokens.revoked = ? AND refresh_tokens.expires_at > ?", userID, false, time.Now()).
		Order("refresh_tokens.created_at DESC").
		Scan(&results).Error

	return results, err
}

func (r *RefreshTokenRepository) RevokeSession(ctx context.Context, userID uuid.UUID, sessionID uint) error {
	return r.DB.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("id = ? AND user_id = ?", sessionID, userID).
		Update("revoked", true).Error
}

func (r *RefreshTokenRepository) RevokeByFingerprint(ctx context.Context, userID uuid.UUID, fingerprint string) error {
	return r.DB.Model(&model.RefreshToken{}).
		Where("user_id = ? AND device_fingerprint = ? AND revoked = ?", userID, fingerprint, false).
		Update("revoked", true).Error
}
