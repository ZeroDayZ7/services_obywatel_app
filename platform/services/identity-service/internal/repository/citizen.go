package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zerodayz7/services/identity-service/db/dbgen"
	"github.com/zerodayz7/services/identity-service/internal/model"
)

var ErrCitizenAlreadyExists = errors.New("citizen already exists")

type CitizenRepository interface {
	CreateWithAudit(ctx context.Context, citizen *model.Citizen, audit *model.CitizenAuditLog) error
	RegisterCitizenWorkflow(ctx context.Context, citizen *model.Citizen, agreement *model.UserAgreement, puk *model.UserPukCode, audit *model.CitizenAuditLog, outbox *model.OutboxMessage) error
	GetByID(ctx context.Context, userID uuid.UUID) (*model.Citizen, error)
	GetByPESELHash(ctx context.Context, peselHash string) (*model.Citizen, error)
}

type citizenRepository struct {
	dbPool *pgxpool.Pool
	q      *dbgen.Queries
}

// #region NewCitizenRepository
func NewCitizenRepository(dbPool *pgxpool.Pool) CitizenRepository {
	return &citizenRepository{
		dbPool: dbPool,
		q:      dbgen.New(dbPool),
	}
}

// #region RegisterCitizenWorkflow
func (r *citizenRepository) RegisterCitizenWorkflow(ctx context.Context, citizen *model.Citizen, agreement *model.UserAgreement, puk *model.UserPukCode, audit *model.CitizenAuditLog, outbox *model.OutboxMessage) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.q.WithTx(tx)

	res, err := qtx.CreateCitizenWithAudit(ctx, dbgen.CreateCitizenWithAuditParams{
		UserID:        citizen.UserID,
		PeselHash:     citizen.PESELHash,
		EncryptedData: citizen.EncryptedData,
		EncryptedDek:  citizen.EncryptedDEK,
		KeyVersion:    citizen.KeyVersion,
	})
	if err != nil {
		return fmt.Errorf("failed to insert citizen: %w", err)
	}
	citizen.CreatedAt = res.CreatedAt

	agreementRow, err := qtx.CreateUserAgreement(ctx, dbgen.CreateUserAgreementParams{
		ID:              agreement.ID,
		UserID:          agreement.UserID,
		AgreementNumber: agreement.AgreementNumber,
		PeselEncrypted:  agreement.PeselEncrypted,
		VerifiedPhone:   agreement.VerifiedPhone,
		Status:          string(agreement.Status),
		SignedAt:        agreement.SignedAt,
		VerifiedAt:      agreement.VerifiedAt,
		VerifiedVia:     agreement.VerifiedVia,
	})
	if err != nil {
		return fmt.Errorf("failed to insert agreement: %w", err)
	}
	agreement.CreatedAt = agreementRow.CreatedAt

	pukRow, err := qtx.CreateUserPukCode(ctx, dbgen.CreateUserPukCodeParams{
		ID:              puk.ID,
		UserAgreementID: agreement.ID,
		UserID:          puk.UserID,
		PukHash:         puk.PukHash,
		Status:          string(puk.Status),
		FailedAttempts:  int16(puk.FailedAttempts),
		MaxAttempts:     int16(puk.MaxAttempts),
		ExpiresAt:       puk.ExpiresAt,
	})
	if err != nil {
		return fmt.Errorf("failed to insert puk code: %w", err)
	}
	puk.CreatedAt = pukRow.CreatedAt

	var prevHash string
	lastLog, err := qtx.GetLastAuditLog(ctx)
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("failed to fetch last audit log: %w", err)
	}
	if err == nil {
		prevHash = lastLog.Hash
	} else {
		prevHash = "0000000000000000000000000000000000000000000000000000000000000000"
	}

	entryHash := calculateAuditHash(audit.ID, audit.UserID, string(audit.Action), audit.ActorID, prevHash)

	err = qtx.CreateAuditLog(ctx, dbgen.CreateAuditLogParams{
		ID:          audit.ID,
		UserID:      audit.UserID,
		Action:      string(audit.Action),
		ActorID:     audit.ActorID,
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

	outboxRow, err := qtx.CreateOutboxMessage(ctx, dbgen.CreateOutboxMessageParams{
		ID:            outbox.ID,
		AggregateType: outbox.AggregateType,
		AggregateID:   outbox.AggregateID,
		EventType:     outbox.EventType,
		Payload:       outbox.Payload,
		Status:        string(outbox.Status),
	})
	if err != nil {
		return fmt.Errorf("failed to insert outbox message: %w", err)
	}
	outbox.CreatedAt = outboxRow.CreatedAt

	return tx.Commit(ctx)
}

// #region CreateWithAudit
func (r *citizenRepository) CreateWithAudit(ctx context.Context, citizen *model.Citizen, audit *model.CitizenAuditLog) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.q.WithTx(tx)

	res, err := qtx.CreateCitizenWithAudit(ctx, dbgen.CreateCitizenWithAuditParams{
		UserID:        citizen.UserID,
		PeselHash:     citizen.PESELHash,
		EncryptedData: citizen.EncryptedData,
		EncryptedDek:  citizen.EncryptedDEK,
		KeyVersion:    citizen.KeyVersion,
	})
	if err != nil {
		return fmt.Errorf("failed to insert citizen: %w", err)
	}
	citizen.CreatedAt = res.CreatedAt

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
		ID:          audit.ID,
		UserID:      audit.UserID,
		Action:      string(audit.Action),
		ActorID:     audit.ActorID,
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

// #region GetByID
func (r *citizenRepository) GetByID(ctx context.Context, userID uuid.UUID) (*model.Citizen, error) {
	row, err := r.q.GetCitizenByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &model.Citizen{
		UserID:        row.UserID,
		PESELHash:     row.PeselHash,
		EncryptedData: row.EncryptedData,
		EncryptedDEK:  row.EncryptedDek,
		KeyVersion:    row.KeyVersion,
		CreatedAt:     row.CreatedAt,
	}, nil
}

// #region GetByPESELHash
func (r *citizenRepository) GetByPESELHash(ctx context.Context, peselHash string) (*model.Citizen, error) {
	row, err := r.q.GetCitizenByPeselHash(ctx, peselHash)
	if err != nil {
		return nil, err
	}

	return &model.Citizen{
		UserID:        row.UserID,
		PESELHash:     row.PeselHash,
		EncryptedData: row.EncryptedData,
		EncryptedDEK:  row.EncryptedDek,
		KeyVersion:    row.KeyVersion,
		CreatedAt:     row.CreatedAt,
	}, nil
}

// #region stringToPgText
func stringToPgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

// #region calculateAuditHash
func calculateAuditHash(id, userID uuid.UUID, action string, actorID uuid.UUID, prevHash string) string {
	raw := fmt.Sprintf("%s:%s:%s:%s:%s", id.String(), userID.String(), action, actorID.String(), prevHash)
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
