package audit

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/audit-service/db/dbgen"
)

type AuditService struct {
	queries dbgen.Querier
	logger  *shared.Logger
}

func NewAuditService(q dbgen.Querier, l *shared.Logger) *AuditService {
	return &AuditService{
		queries: q,
		logger:  l,
	}
}

// SaveLog teraz przyjmuje strukturę AuditMessage
func (s *AuditService) SaveLog(ctx context.Context, msg AuditMessage) error {
	metadataBytes, err := json.Marshal(msg.Metadata)
	if err != nil {
		s.logger.ErrorObj("Failed to marshal metadata", err)
		return err
	}

	uid, err := toUUID(msg.UserID)
	if err != nil {
		s.logger.ErrorObj("Invalid UUID", err)
		return err
	}

	// Używamy string, bo w sqlc typ dla JSONB to string
	err = s.queries.CreateLog(ctx, dbgen.CreateLogParams{
		UserID:      uid,
		ServiceName: msg.Service,
		Action:      msg.Action,
		IpAddress:   msg.IPAddress,
		Metadata:    metadataBytes,
		Status:      "SUCCESS",
	})
	if err != nil {
		s.logger.ErrorObj("Failed to save log to DB", err)
		return err
	}

	return nil
}

// --- Metody dla Handlera (Odczyt zaszyfrowanych danych) ---

func (s *AuditService) GetAllLogs(ctx context.Context, limit, offset int32) ([]dbgen.AuditLog, error) {
	return s.queries.GetAllLogs(ctx, dbgen.GetAllLogsParams{
		Limit:  limit,
		Offset: offset,
	})
}

func (s *AuditService) GetLogsByAction(ctx context.Context, action string) ([]dbgen.AuditLog, error) {
	return s.queries.GetLogsByAction(ctx, action)
}

func (s *AuditService) GetLogsByUserID(ctx context.Context, userID uuid.UUID) ([]dbgen.AuditLog, error) {
	uid, err := toUUID(userID.String())
	if err != nil {
		s.logger.ErrorObj("Invalid UUID", err)
		return nil, err
	}

	return s.queries.GetLogsByUserId(ctx, uid)
}

func (s *AuditService) GetLogByID(ctx context.Context, id int64) (dbgen.AuditLog, error) {
	return s.queries.GetLogByID(ctx, id)
}

func (s *AuditService) SaveLogsBatch(ctx context.Context, logs []AuditMessage) error {
	for _, log := range logs {
		if err := s.SaveLog(ctx, log); err != nil {
			return err
		}
	}
	return nil
}

// Pomocnik do konwersji string -> pgtype.UUID
func toUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return pgtype.UUID{}, err
	}
	if !u.Valid {
		return pgtype.UUID{}, errors.New("invalid UUID")
	}
	return u, nil
}
