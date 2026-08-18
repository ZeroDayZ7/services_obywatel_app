package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zerodayz7/services/identity-service/internal/model"
)

type CitizenRepository interface {
	Create(ctx context.Context, citizen *model.Citizen) error
	GetByID(ctx context.Context, userID uuid.UUID) (*model.Citizen, error)
	GetByPESELHash(ctx context.Context, peselHash string) (*model.Citizen, error)
}

type citizenRepository struct {
	db *pgxpool.Pool
}

func NewCitizenRepository(db *pgxpool.Pool) CitizenRepository {
	return &citizenRepository{db: db}
}

func (r *citizenRepository) Create(ctx context.Context, citizen *model.Citizen) error {
	query := `
		INSERT INTO citizens (user_id, pesel_hash, encrypted_data, encrypted_dek, nonce, key_version)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Exec(ctx, query,
		citizen.UserID,
		citizen.PESELHash,
		citizen.EncryptedData,
		citizen.EncryptedDEK,
		citizen.Nonce,
		citizen.KeyVersion,
	)
	return err
}

func (r *citizenRepository) GetByID(ctx context.Context, userID uuid.UUID) (*model.Citizen, error) {
	query := `
		SELECT user_id, pesel_hash, encrypted_data, encrypted_dek, nonce, key_version, created_at, updated_at
		FROM citizens WHERE user_id = $1
	`
	c := &model.Citizen{}
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&c.UserID, &c.PESELHash, &c.EncryptedData, &c.EncryptedDEK, &c.Nonce, &c.KeyVersion, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (r *citizenRepository) GetByPESELHash(ctx context.Context, peselHash string) (*model.Citizen, error) {
	query := `
		SELECT user_id, pesel_hash, encrypted_data, encrypted_dek, nonce, key_version, created_at, updated_at
		FROM citizens WHERE pesel_hash = $1
	`
	c := &model.Citizen{}
	err := r.db.QueryRow(ctx, query, peselHash).Scan(
		&c.UserID, &c.PESELHash, &c.EncryptedData, &c.EncryptedDEK, &c.Nonce, &c.KeyVersion, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return c, nil
}
