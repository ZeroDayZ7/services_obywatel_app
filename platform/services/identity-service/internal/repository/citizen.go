package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zerodayz7/services/identity-service/db/dbgen"
	"github.com/zerodayz7/services/identity-service/internal/model"
)

type CitizenRepository interface {
	CreateWithAudit(ctx context.Context, citizen *model.Citizen, audit *model.CitizenAuditLog) error
	GetByID(ctx context.Context, userID uuid.UUID) (*model.Citizen, error)
	GetByPESELHash(ctx context.Context, peselHash string) (*model.Citizen, error)
}

type citizenRepository struct {
	dbPool *pgxpool.Pool
	q      *dbgen.Queries
}

func NewCitizenRepository(dbPool *pgxpool.Pool) CitizenRepository {
	return &citizenRepository{
		dbPool: dbPool,
		q:      dbgen.New(dbPool),
	}
}

func (r *citizenRepository) CreateWithAudit(ctx context.Context, citizen *model.Citizen, audit *model.CitizenAuditLog) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.q.WithTx(tx)

	res, err := qtx.CreateCitizenWithAudit(ctx, dbgen.CreateCitizenWithAuditParams{
		UserID:        uuidToPgType(citizen.UserID),
		PeselHash:     citizen.PESELHash,
		EncryptedData: citizen.EncryptedData,
		EncryptedDek:  citizen.EncryptedDEK,
		KeyVersion:    citizen.KeyVersion,
	})
	if err != nil {
		return fmt.Errorf("failed to insert citizen: %w", err)
	}
	citizen.CreatedAt = res.CreatedAt.Time

	var prevHash string
	lastLog, err := qtx.GetLastAuditLog(ctx)
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("failed to fetch last audit log for hashing: %w", err)
	}
	if err == nil {
		prevHash = lastLog.Hash
	} else {
		prevHash = "0000000000000000000000000000000000000000000000000000000000000000"
	}

	entryHash := calculateAuditHash(audit.ID, audit.UserID, string(audit.Action), audit.ActorID, prevHash)

	err = qtx.CreateAuditLog(ctx, dbgen.CreateAuditLogParams{
		ID:          uuidToPgType(audit.ID),
		UserID:      uuidToPgType(audit.UserID),
		Action:      string(audit.Action),
		ActorID:     uuidToPgType(audit.ActorID),
		IpAddress:   stringToPgText(audit.IPAddress),
		PayloadHash: stringToPgText(audit.PayloadHash),
		PrevHash:    prevHash,
		Hash:        entryHash,
	})
	if err != nil {
		return fmt.Errorf("failed to insert audit log: %w", err)
	}

	audit.PrevHash = prevHash
	audit.Hash = entryHash

	return tx.Commit(ctx)
}

func (r *citizenRepository) GetByID(ctx context.Context, userID uuid.UUID) (*model.Citizen, error) {
	row, err := r.q.GetCitizenByUserID(ctx, uuidToPgType(userID))
	if err != nil {
		return nil, err
	}

	return &model.Citizen{
		UserID:        row.UserID.Bytes,
		PESELHash:     row.PeselHash,
		EncryptedData: row.EncryptedData,
		EncryptedDEK:  row.EncryptedDek,
		KeyVersion:    row.KeyVersion,
		CreatedAt:     row.CreatedAt.Time,
	}, nil
}

func (r *citizenRepository) GetByPESELHash(ctx context.Context, peselHash string) (*model.Citizen, error) {
	row, err := r.q.GetCitizenByPeselHash(ctx, peselHash)
	if err != nil {
		return nil, err
	}

	return &model.Citizen{
		UserID:        row.UserID.Bytes,
		PESELHash:     row.PeselHash,
		EncryptedData: row.EncryptedData,
		EncryptedDEK:  row.EncryptedDek,
		KeyVersion:    row.KeyVersion,
		CreatedAt:     row.CreatedAt.Time,
	}, nil
}

// Funkcje pomocnicze do mapowania typów Go <-> pgx/v5

func uuidToPgType(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: u, Valid: true}
}

func stringToPgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func calculateAuditHash(id, userID uuid.UUID, action string, actorID uuid.UUID, prevHash string) string {
	raw := fmt.Sprintf("%s:%s:%s:%s:%s", id.String(), userID.String(), action, actorID.String(), prevHash)
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
