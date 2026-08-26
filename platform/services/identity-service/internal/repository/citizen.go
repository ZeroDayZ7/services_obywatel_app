package repository

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zerodayz7/services/identity-service/db/dbgen"
	"github.com/zerodayz7/services/identity-service/internal/model"
)

const zeroAuditHash = "0000000000000000000000000000000000000000000000000000000000000000"

type txKey struct{}

type citizenRepository struct {
	dbPool       *pgxpool.Pool
	q            *dbgen.Queries
	auditHmacKey []byte
}

// #region NewCitizenRepository
//#region NewCitizenRepository
func NewCitizenRepository(dbPool *pgxpool.Pool, auditHmacKey []byte) CitizenRepository {
	return &citizenRepository{
		dbPool:       dbPool,
		q:            dbgen.New(dbPool),
		auditHmacKey: auditHmacKey,
	}
}

// #endregion NewCitizenRepository

// #region WithinTx
//#region WithinTx
func (r *citizenRepository) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := r.dbPool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	qtx := r.q.WithTx(tx)
	txCtx := context.WithValue(ctx, txKey{}, qtx)

	if err := fn(txCtx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// #endregion WithinTx

// #region getQueries
//#region getQueries
func (r *citizenRepository) getQueries(ctx context.Context) *dbgen.Queries {
	if q, ok := ctx.Value(txKey{}).(*dbgen.Queries); ok {
		return q
	}
	return r.q
}

// #endregion getQueries

// #region Create
//#region Create
func (r *citizenRepository) Create(ctx context.Context, citizen *model.Citizen) error {
	q := r.getQueries(ctx)
	res, err := q.CreateCitizenWithAudit(ctx, dbgen.CreateCitizenWithAuditParams{
		UserID:        citizen.UserID,
		PeselHash:     citizen.PESELHash,
		EmailHash:     citizen.EmailHash,
		PhoneHash:     citizen.PhoneHash,
		EncryptedData: citizen.EncryptedData,
		EncryptedDek:  citizen.EncryptedDEK,
	})
	if err != nil {
		return fmt.Errorf("failed to insert citizen: %w", err)
	}
	citizen.CreatedAt = res.CreatedAt
	return nil
}

// #endregion Create

// #region CreateAgreement
//#region CreateAgreement
func (r *citizenRepository) CreateAgreement(ctx context.Context, agreement *model.UserAgreement) error {
	q := r.getQueries(ctx)
	row, err := q.CreateUserAgreement(ctx, dbgen.CreateUserAgreementParams{
		ID:              agreement.ID,
		UserID:          agreement.UserID,
		AgreementNumber: agreement.AgreementNumber,
		S3Key:           agreement.S3Key,
		S3Bucket:        agreement.S3Bucket,
		EncryptedDek:    agreement.EncryptedDEK,
		EncryptedEmail:  agreement.EncryptedEmail,
		EncryptedPhone:  agreement.EncryptedPhone,
		Status:          string(agreement.Status),
		SignedAt:        agreement.SignedAt,
		VerifiedAt:      agreement.VerifiedAt,
		VerifiedVia:     agreement.VerifiedVia,
	})
	if err != nil {
		return fmt.Errorf("failed to insert agreement: %w", err)
	}
	agreement.CreatedAt = row.CreatedAt
	return nil
}

// #endregion CreateAgreement

// #region CreatePukCode
//#region CreatePukCode
func (r *citizenRepository) CreatePukCode(ctx context.Context, puk *model.UserPukCode) error {
	q := r.getQueries(ctx)
	row, err := q.CreateUserPukCode(ctx, dbgen.CreateUserPukCodeParams{
		ID:              puk.ID,
		UserAgreementID: puk.UserAgreementID,
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
	puk.CreatedAt = row.CreatedAt
	return nil
}

// #endregion CreatePukCode

// #region CreateAuditLog
//#region CreateAuditLog
func (r *citizenRepository) CreateAuditLog(ctx context.Context, audit *model.CitizenAuditLog) error {
	q := r.getQueries(ctx)

	prevHash := zeroAuditHash
	lastLog, err := q.GetLastAuditLog(ctx)
	if err == nil {
		prevHash = lastLog.Hash
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to fetch last audit log: %w", err)
	}

	audit.PrevHash = prevHash
	audit.Hash = calculateAuditHMAC(
		audit.ID,
		audit.UserID,
		string(audit.Action),
		audit.ActorID,
		audit.IPAddress,
		audit.PayloadHash,
		prevHash,
		r.auditHmacKey,
	)

	err = q.CreateAuditLog(ctx, dbgen.CreateAuditLogParams{
		ID:          audit.ID,
		UserID:      audit.UserID,
		Action:      string(audit.Action),
		ActorID:     audit.ActorID,
		IpAddress:   audit.IPAddress,
		PayloadHash: audit.PayloadHash,
		PrevHash:    audit.PrevHash,
		Hash:        audit.Hash,
	})
	if err != nil {
		return fmt.Errorf("failed to insert audit log: %w", err)
	}
	return nil
}

// #endregion CreateAuditLog

// #region CreateOutboxMessage
//#region CreateOutboxMessage
func (r *citizenRepository) CreateOutboxMessage(ctx context.Context, outbox *model.OutboxMessage) error {
	q := r.getQueries(ctx)
	row, err := q.CreateOutboxMessage(ctx, dbgen.CreateOutboxMessageParams{
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
	outbox.CreatedAt = row.CreatedAt
	return nil
}

// #endregion CreateOutboxMessage

// #region GetByID
//#region GetByID
func (r *citizenRepository) GetByID(ctx context.Context, userID uuid.UUID) (*model.Citizen, error) {
	q := r.getQueries(ctx)
	row, err := q.GetCitizenByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &model.Citizen{
		UserID:        row.UserID,
		PESELHash:     row.PeselHash,
		EncryptedData: row.EncryptedData,
		EncryptedDEK:  row.EncryptedDek,
		CreatedAt:     row.CreatedAt,
	}, nil
}

// #endregion GetByID

// #region GetByPESELHash
//#region GetByPESELHash
func (r *citizenRepository) GetByPESELHash(ctx context.Context, peselHash string) (*model.Citizen, error) {
	q := r.getQueries(ctx)
	row, err := q.GetCitizenByPeselHash(ctx, peselHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &model.Citizen{
		UserID:        row.UserID,
		PESELHash:     row.PeselHash,
		EmailHash:     row.EmailHash,
		PhoneHash:     row.PhoneHash,
		EncryptedData: row.EncryptedData,
		EncryptedDEK:  row.EncryptedDek,
		CreatedAt:     row.CreatedAt,
	}, nil
}

// #endregion GetByPESELHash

// #region GetByEmailHash
//#region GetByEmailHash
func (r *citizenRepository) GetByEmailHash(ctx context.Context, emailHash string) (*model.Citizen, error) {
	q := r.getQueries(ctx)
	row, err := q.GetCitizenByEmailHash(ctx, emailHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &model.Citizen{
		UserID:        row.UserID,
		PESELHash:     row.PeselHash,
		EmailHash:     row.EmailHash,
		PhoneHash:     row.PhoneHash,
		EncryptedData: row.EncryptedData,
		EncryptedDEK:  row.EncryptedDek,
		CreatedAt:     row.CreatedAt,
	}, nil
}

// #endregion GetByEmailHash

// #region GetByPhoneHash
//#region GetByPhoneHash
func (r *citizenRepository) GetByPhoneHash(ctx context.Context, phoneHash string) (*model.Citizen, error) {
	q := r.getQueries(ctx)
	row, err := q.GetCitizenByPhoneHash(ctx, phoneHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &model.Citizen{
		UserID:        row.UserID,
		PESELHash:     row.PeselHash,
		EmailHash:     row.EmailHash,
		PhoneHash:     row.PhoneHash,
		EncryptedData: row.EncryptedData,
		EncryptedDEK:  row.EncryptedDek,
		CreatedAt:     row.CreatedAt,
	}, nil
}

// #endregion GetByPhoneHash

// #region GetAgreementByID
//#region GetAgreementByID
func (r *citizenRepository) GetAgreementByID(ctx context.Context, agreementID uuid.UUID) (*model.UserAgreement, error) {
	q := r.getQueries(ctx)
	row, err := q.GetAgreementByID(ctx, agreementID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &model.UserAgreement{
		ID:              row.ID,
		UserID:          row.UserID,
		AgreementNumber: row.AgreementNumber,
		S3Key:           row.S3Key,
		S3Bucket:        row.S3Bucket,
		EncryptedDEK:    row.EncryptedDek,
		EncryptedEmail:  row.EncryptedEmail,
		EncryptedPhone:  row.EncryptedPhone,
		Status:          model.AgreementStatus(row.Status),
		SignedAt:        row.SignedAt,
		VerifiedAt:      row.VerifiedAt,
		VerifiedVia:     row.VerifiedVia,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}, nil
}

// #endregion GetAgreementByID

// #region calculateAuditHMAC
//#region calculateAuditHMAC
func calculateAuditHMAC(id, userID uuid.UUID, action string, actorID uuid.UUID, ipAddress, payloadHash, prevHash string, hmacSecret []byte) string {
	// Łączymy wszystkie kluczowe pola (w tym IPAddress i PrevHash)
	raw := fmt.Sprintf("%s:%s:%s:%s:%s:%s:%s",
		id.String(),
		userID.String(),
		action,
		actorID.String(),
		ipAddress,
		payloadHash,
		prevHash,
	)

	h := hmac.New(sha256.New, hmacSecret)
	h.Write([]byte(raw))
	return hex.EncodeToString(h.Sum(nil))
}

// #endregion calculateAuditHMAC
