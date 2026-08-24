package worker

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zerodayz7/platform/pkg/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	defaultAuditWorkerBatchSize     = 200
	defaultAuditWorkerInterval      = 2 * time.Second
	defaultAuditWorkerMaxRetries    = 10
	defaultAuditWorkerBackoffBase   = time.Second
	defaultAuditWorkerBackoffMax    = 60 * time.Second
	defaultAuditWorkerConcurrency   = 1
	defaultAuditWorkerRoutingKey    = "audit.log.created"
	defaultAuditWorkerSourceService = "identity-service"
)

type AuditEvent struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Action      string
	ActorID     uuid.UUID
	IPAddress   string
	PayloadHash string
	PrevHash    string
	Hash        string
	CreatedAt   time.Time
}

type AuditWorkerConfig struct {
	Enabled       bool
	BatchSize     int
	Interval      time.Duration
	MaxRetries    int
	BackoffBase   time.Duration
	BackoffMax    time.Duration
	Concurrency   int
	RoutingKey    string
	SourceService string
}

type AuditRepository interface {
	FetchPending(ctx context.Context, limit int, leaseWindow time.Duration) ([]AuditEvent, error)
	MarkSynced(ctx context.Context, ids []uuid.UUID) error
	MarkRetry(ctx context.Context, id uuid.UUID, retryCount int, reason string) error
	MarkFailed(ctx context.Context, id uuid.UUID, retryCount int, reason string) error
	PendingStats(ctx context.Context) (pending int, oldestAge time.Duration, err error)
}

type AuditPublisher interface {
	Publish(ctx context.Context, routingKey string, payload []byte) error
	Close() error
}

type AuditWorker struct {
	repo      AuditRepository
	publisher AuditPublisher
	cfg       AuditWorkerConfig
	log       *shared.Logger
	tracer    trace.Tracer
}

func normalizeAuditWorkerConfig(cfg AuditWorkerConfig) AuditWorkerConfig {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = defaultAuditWorkerBatchSize
	}
	if cfg.Interval <= 0 {
		cfg.Interval = defaultAuditWorkerInterval
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = defaultAuditWorkerMaxRetries
	}
	if cfg.BackoffBase <= 0 {
		cfg.BackoffBase = defaultAuditWorkerBackoffBase
	}
	if cfg.BackoffMax <= 0 {
		cfg.BackoffMax = defaultAuditWorkerBackoffMax
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = defaultAuditWorkerConcurrency
	}
	if cfg.RoutingKey == "" {
		cfg.RoutingKey = defaultAuditWorkerRoutingKey
	}
	if cfg.SourceService == "" {
		cfg.SourceService = defaultAuditWorkerSourceService
	}
	return cfg
}

func NewAuditWorker(db *pgxpool.Pool, publisher AuditPublisher, cfg AuditWorkerConfig) *AuditWorker {
	return NewAuditWorkerWithDeps(newPostgresAuditRepository(db), publisher, cfg)
}

func NewAuditWorkerWithDeps(repo AuditRepository, publisher AuditPublisher, cfg AuditWorkerConfig) *AuditWorker {
	cfg = normalizeAuditWorkerConfig(cfg)
	return &AuditWorker{
		repo:      repo,
		publisher: publisher,
		cfg:       cfg,
		log:       shared.GetLogger(),
		tracer:    otel.Tracer("identity-service.audit-worker"),
	}
}

func (w *AuditWorker) Start(ctx context.Context) {
	if w.repo == nil || w.publisher == nil {
		w.log.Error("audit worker cannot start without repository or publisher")
		return
	}

	ticker := time.NewTicker(w.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Info("audit worker shutdown requested")
			return
		case <-ticker.C:
			if err := w.processOnce(ctx); err != nil {
				w.log.Error("audit worker process iteration failed", "error", err)
			}
		}
	}
}

func (w *AuditWorker) ProcessOnce(ctx context.Context) error {
	return w.processOnce(ctx)
}

func (w *AuditWorker) processOnce(ctx context.Context) error {
	ctx, span := w.tracer.Start(ctx, "audit_worker.process_once")
	defer span.End()

	batch, err := w.repo.FetchPending(ctx, w.cfg.BatchSize, 5*time.Minute)
	if err != nil {
		span.RecordError(err)
		return err
	}
	if len(batch) == 0 {
		return nil
	}

	batchID := uuid.New()
	sem := make(chan struct{}, max(1, w.cfg.Concurrency))
	errCh := make(chan error, len(batch))
	var wg sync.WaitGroup

	for _, event := range batch {
		event := event
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if err := w.publishWithRetry(ctx, batchID, event); err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil && !errors.Is(err, context.Canceled) {
			w.log.Error("audit event publish failed in batch", "batch_id", batchID, "error", err)
		}
	}
	return nil
}

func (w *AuditWorker) publishWithRetry(ctx context.Context, batchID uuid.UUID, event AuditEvent) error {
	for attempt := 0; attempt <= w.cfg.MaxRetries; attempt++ {
		if err := w.publishSingle(ctx, batchID, event); err != nil {
			if !isRetryableAuditError(err) {
				if markErr := w.repo.MarkFailed(ctx, event.ID, attempt+1, err.Error()); markErr != nil {
					w.log.Error("failed to mark permanent audit error", "event_id", event.ID, "error", markErr)
				}
				return err
			}
			if attempt >= w.cfg.MaxRetries {
				if markErr := w.repo.MarkFailed(ctx, event.ID, attempt+1, err.Error()); markErr != nil {
					w.log.Error("failed to mark audit event after max retries", "event_id", event.ID, "error", markErr)
				}
				return err
			}
			if retryErr := w.repo.MarkRetry(ctx, event.ID, attempt+1, err.Error()); retryErr != nil {
				w.log.Error("failed to update retry state for audit event", "event_id", event.ID, "error", retryErr)
			}
			delay := computeBackoff(attempt, w.cfg.BackoffBase, w.cfg.BackoffMax)
			w.log.Warn("audit publish retry scheduled", "event_id", event.ID, "attempt", attempt+1, "delay", delay)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			continue
		}

		if err := w.repo.MarkSynced(ctx, []uuid.UUID{event.ID}); err != nil {
			w.log.Error("failed to mark audit event as synced", "event_id", event.ID, "error", err)
			return err
		}
		return nil
	}
	return nil
}

func (w *AuditWorker) publishSingle(ctx context.Context, batchID uuid.UUID, event AuditEvent) error {
	ctx, span := w.tracer.Start(ctx, "audit_worker.publish")
	defer span.End()
	span.SetAttributes(
		attribute.String("event_id", event.ID.String()),
		attribute.String("batch_id", batchID.String()),
	)

	payload, err := buildAuditEnvelope(event, batchID, w.cfg.SourceService)
	if err != nil {
		span.RecordError(err)
		return err
	}

	if err := w.publisher.Publish(ctx, w.cfg.RoutingKey, payload); err != nil {
		span.RecordError(err)
		return err
	}
	return nil
}

func buildAuditEnvelope(event AuditEvent, batchID uuid.UUID, sourceService string) ([]byte, error) {
	payload := map[string]any{
		"message_id":     uuid.NewString(),
		"batch_id":       batchID.String(),
		"event_id":       event.ID.String(),
		"event_type":     event.Action,
		"schema_version": 1,
		"source_service": sourceService,
		"created_at":     event.CreatedAt.UTC().Format(time.RFC3339Nano),
		"trace_id":       uuid.NewString(),
		"payload": map[string]any{
			"user_id":      event.UserID.String(),
			"actor_id":     event.ActorID.String(),
			"action":       event.Action,
			"ip_address":   strings.TrimSpace(event.IPAddress),
			"payload_hash": event.PayloadHash,
			"prev_hash":    event.PrevHash,
			"hash":         event.Hash,
		},
	}
	return json.Marshal(payload)
}

func isRetryableAuditError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, token := range []string{
		"timeout",
		"temporarily unavailable",
		"temporary",
		"temporary network",
		"network issue",
		"connection reset",
		"broken pipe",
		"refused",
		"nack",
		"unroutable",
		"no route",
		"econnrefused",
		"connection closed",
		"try again",
	} {
		if strings.Contains(msg, token) {
			return true
		}
	}
	return false
}

func computeBackoff(attempt int, base, max time.Duration) time.Duration {
	if base <= 0 {
		base = time.Second
	}
	if max <= 0 {
		max = 60 * time.Second
	}
	delay := time.Duration(1<<attempt) * base
	if delay > max {
		delay = max
	}
	if attempt > 0 {
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		jitter := time.Duration(rng.Int63n(int64(base)))
		delay += jitter
	}
	if delay > max {
		delay = max
	}
	return delay
}

type postgresAuditRepository struct {
	db *pgxpool.Pool
}

func newPostgresAuditRepository(db *pgxpool.Pool) *postgresAuditRepository {
	return &postgresAuditRepository{db: db}
}

func (r *postgresAuditRepository) FetchPending(ctx context.Context, limit int, leaseWindow time.Duration) ([]AuditEvent, error) {
	if r.db == nil {
		return nil, errors.New("database pool is nil")
	}
	if limit <= 0 {
		limit = defaultAuditWorkerBatchSize
	}
	if leaseWindow <= 0 {
		leaseWindow = 5 * time.Minute
	}

	rows, err := r.db.Query(ctx, `
		WITH claimed AS (
			SELECT id, user_id, action, actor_id, ip_address, payload_hash, prev_hash, hash, created_at
			FROM citizen_audit_logs
			WHERE synced_to_global_audit = FALSE
			  AND (sync_state IS NULL OR sync_state IN ('PENDING','RETRY'))
			  AND (processing_started_at IS NULL OR processing_started_at < NOW() - $2::interval)
			ORDER BY created_at ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE citizen_audit_logs a
		SET sync_state = 'PROCESSING', processing_started_at = NOW(), updated_at = NOW()
		FROM claimed c
		WHERE a.id = c.id
		RETURNING a.id, a.user_id, a.action, a.actor_id, a.ip_address, a.payload_hash, a.prev_hash, a.hash, a.created_at`,
		limit,
		leaseWindow.String(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []AuditEvent
	for rows.Next() {
		var e AuditEvent
		if err := rows.Scan(&e.ID, &e.UserID, &e.Action, &e.ActorID, &e.IPAddress, &e.PayloadHash, &e.PrevHash, &e.Hash, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (r *postgresAuditRepository) MarkSynced(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.db.Exec(ctx, `
		UPDATE citizen_audit_logs
		SET synced_to_global_audit = TRUE,
		    sync_state = 'SENT',
		    processed_at = NOW(),
		    updated_at = NOW(),
		    last_error = NULL,
		    processing_started_at = NULL
		WHERE id = ANY($1::uuid[])`, ids)
	return err
}

func (r *postgresAuditRepository) MarkRetry(ctx context.Context, id uuid.UUID, retryCount int, reason string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE citizen_audit_logs
		SET sync_state = 'RETRY',
		    retry_count = $2,
		    last_error = $3,
		    updated_at = NOW(),
		    processing_started_at = NULL
		WHERE id = $1`, id, retryCount, reason)
	return err
}

func (r *postgresAuditRepository) MarkFailed(ctx context.Context, id uuid.UUID, retryCount int, reason string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE citizen_audit_logs
		SET sync_state = 'FAILED',
		    retry_count = $2,
		    last_error = $3,
		    updated_at = NOW(),
		    processing_started_at = NULL,
		    processed_at = NOW()
		WHERE id = $1`, id, retryCount, reason)
	return err
}

func (r *postgresAuditRepository) PendingStats(ctx context.Context) (pending int, oldestAge time.Duration, err error) {
	row := r.db.QueryRow(ctx, `
		SELECT COUNT(*)::int,
		       COALESCE(EXTRACT(EPOCH FROM (NOW() - MIN(created_at))), 0.0)
		FROM citizen_audit_logs
		WHERE synced_to_global_audit = FALSE AND (sync_state IS NULL OR sync_state IN ('PENDING','RETRY','PROCESSING'))`)
	var ageSeconds float64
	if err = row.Scan(&pending, &ageSeconds); err != nil {
		return 0, 0, err
	}
	return pending, time.Duration(ageSeconds * float64(time.Second)), nil
}

func (w *AuditWorker) Close() error {
	if w.publisher != nil {
		return w.publisher.Close()
	}
	return nil
}

func (w *AuditWorker) Shutdown() error {
	return w.Close()
}

type noopAuditPublisher struct{}

func (n *noopAuditPublisher) Publish(ctx context.Context, routingKey string, payload []byte) error {
	return nil
}

func (n *noopAuditPublisher) Close() error { return nil }

var (
	_ AuditRepository = (*postgresAuditRepository)(nil)
	_ AuditPublisher  = (*noopAuditPublisher)(nil)
)
