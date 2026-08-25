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
	"github.com/zerodayz7/platform/pkg/rabbitmq"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/services/identity-service/internal/model"
	"github.com/zerodayz7/services/identity-service/internal/repository"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	defaultRegistrationWorkerBatchSize   = 100
	defaultRegistrationWorkerInterval    = 2 * time.Second
	defaultRegistrationWorkerMaxRetries  = 5
	defaultRegistrationWorkerBackoffBase = time.Second
	defaultRegistrationWorkerBackoffMax  = 30 * time.Second
	defaultRegistrationWorkerConcurrency = 2
	defaultRegistrationWorkerRoutingKey  = "auth.register"
)

type RegistrationWorkerConfig struct {
	Enabled     bool
	BatchSize   int
	Interval    time.Duration
	MaxRetries  int
	BackoffBase time.Duration
	BackoffMax  time.Duration
	Concurrency int
	RoutingKey  string
}

type RegistrationPublisher interface {
	Publish(ctx context.Context, routingKey string, payload []byte) error
	Close() error
}

type RegistrationWorker struct {
	repo      repository.OutboxRepository
	publisher RegistrationPublisher
	cfg       RegistrationWorkerConfig
	log       *shared.Logger
	tracer    trace.Tracer
}

//#region normalizeRegistrationConfig
func normalizeRegistrationConfig(cfg RegistrationWorkerConfig) RegistrationWorkerConfig {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = defaultRegistrationWorkerBatchSize
	}
	if cfg.Interval <= 0 {
		cfg.Interval = defaultRegistrationWorkerInterval
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = defaultRegistrationWorkerMaxRetries
	}
	if cfg.BackoffBase <= 0 {
		cfg.BackoffBase = defaultRegistrationWorkerBackoffBase
	}
	if cfg.BackoffMax <= 0 {
		cfg.BackoffMax = defaultRegistrationWorkerBackoffMax
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = defaultRegistrationWorkerConcurrency
	}
	if cfg.RoutingKey == "" {
		cfg.RoutingKey = defaultRegistrationWorkerRoutingKey
	}
	return cfg
}

//#region NewRegistrationWorker
func NewRegistrationWorker(db *pgxpool.Pool, publisher RegistrationPublisher, cfg RegistrationWorkerConfig) *RegistrationWorker {
	return NewRegistrationWorkerWithDeps(repository.NewOutboxRepository(db), publisher, cfg)
}

//#region NewRegistrationWorkerWithDeps
func NewRegistrationWorkerWithDeps(repo repository.OutboxRepository, publisher RegistrationPublisher, cfg RegistrationWorkerConfig) *RegistrationWorker {
	cfg = normalizeRegistrationConfig(cfg)
	return &RegistrationWorker{
		repo:      repo,
		publisher: publisher,
		cfg:       cfg,
		log:       shared.GetLogger(),
		tracer:    otel.Tracer("identity-service.registration-worker"),
	}
}

//#region Start
func (w *RegistrationWorker) Start(ctx context.Context) {
	if w.repo == nil || w.publisher == nil {
		w.log.Error("registration worker cannot start without repository or publisher")
		return
	}

	ticker := time.NewTicker(w.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Info("registration worker shutdown requested")
			return
		case <-ticker.C:
			if err := w.processOnce(ctx); err != nil {
				w.log.Error("registration worker process iteration failed", "error", err)
			}
		}
	}
}

//#region ProcessOnce
func (w *RegistrationWorker) ProcessOnce(ctx context.Context) error {
	return w.processOnce(ctx)
}

//#region processOnce
func (w *RegistrationWorker) processOnce(ctx context.Context) error {
	ctx, span := w.tracer.Start(ctx, "registration_worker.process_once")
	defer span.End()

	batch, err := w.repo.FetchPendingMessages(ctx, w.cfg.BatchSize)
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

	for _, msg := range batch {
		m := msg
		if strings.HasPrefix(m.EventType, "citizen.") {
			wg.Add(1)
			go func() {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				if err := w.publishWithRetry(ctx, batchID, m); err != nil {
					errCh <- err
				}
			}()
		}
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil && !errors.Is(err, context.Canceled) {
			w.log.Error("registration message publish failed in batch", "batch_id", batchID, "error", err)
		}
	}
	return nil
}

//#region publishWithRetry
func (w *RegistrationWorker) publishWithRetry(ctx context.Context, batchID uuid.UUID, msg model.OutboxMessage) error {
	for attempt := 0; attempt <= w.cfg.MaxRetries; attempt++ {
		if err := w.publishSingle(ctx, batchID, msg); err != nil {
			if !isRetryableRegistrationError(err) {
				if markErr := w.repo.MarkAsFailed(ctx, msg.ID, int16(w.cfg.MaxRetries), err.Error()); markErr != nil {
					w.log.Error("failed to mark registration message permanent failure", "msg_id", msg.ID, "error", markErr)
				}
				return err
			}
			if attempt >= w.cfg.MaxRetries {
				if markErr := w.repo.MarkAsFailed(ctx, msg.ID, int16(w.cfg.MaxRetries), err.Error()); markErr != nil {
					w.log.Error("failed to mark registration message after max retries", "msg_id", msg.ID, "error", markErr)
				}
				return err
			}
			if retryErr := w.repo.MarkAsFailed(ctx, msg.ID, int16(attempt+1), err.Error()); retryErr != nil {
				w.log.Error("failed to update retry state for registration message", "msg_id", msg.ID, "error", retryErr)
			}
			delay := computeRegistrationBackoff(attempt, w.cfg.BackoffBase, w.cfg.BackoffMax)
			w.log.Warn("registration publish retry scheduled", "msg_id", msg.ID, "attempt", attempt+1, "delay", delay)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			continue
		}

		if err := w.repo.MarkAsSent(ctx, msg.ID); err != nil {
			w.log.Error("failed to mark registration message as sent", "msg_id", msg.ID, "error", err)
			return err
		}
		return nil
	}
	return nil
}

//#region publishSingle
func (w *RegistrationWorker) publishSingle(ctx context.Context, batchID uuid.UUID, msg model.OutboxMessage) error {
	ctx, span := w.tracer.Start(ctx, "registration_worker.publish")
	defer span.End()
	span.SetAttributes(
		attribute.String("message_id", msg.ID.String()),
		attribute.String("batch_id", batchID.String()),
	)

	var payload any
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		payload = map[string]any{"raw_payload": string(msg.Payload)}
	}

	envelope := map[string]any{
		"message_id":     uuid.NewString(),
		"batch_id":       batchID.String(),
		"outbox_id":      msg.ID.String(),
		"aggregate_type": msg.AggregateType,
		"aggregate_id":   msg.AggregateID.String(),
		"event_type":     msg.EventType,
		"schema_version": 1,
		"created_at":     msg.CreatedAt.UTC().Format(time.RFC3339Nano),
		"payload":        payload,
	}

	body, err := json.Marshal(envelope)
	if err != nil {
		span.RecordError(err)
		return err
	}

	// Używamy event_type bezpośrednio z bazy jako Routing Key,
	// a jeśli w bazie byłby pusty, bierzemy fallback z konfiguracji lub pakietu rabbitmq
	routingKey := msg.EventType
	if routingKey == "" {
		routingKey = rabbitmq.TopicCitizenCreated
	}

	if err := w.publisher.Publish(ctx, routingKey, body); err != nil {
		span.RecordError(err)
		return err
	}
	return nil
}

//#region Close
func (w *RegistrationWorker) Close() error {
	if w.publisher != nil {
		return w.publisher.Close()
	}
	return nil
}

//#region isRetryableRegistrationError
func isRetryableRegistrationError(err error) bool {
	if err == nil {
		return false
	}
	// Możesz rozbudować o błędy sieciowe/AMQP, domyślnie przyjmujemy wszystko za powtarzalne
	return true
}

//#region computeRegistrationBackoff
func computeRegistrationBackoff(attempt int, base, maxVal time.Duration) time.Duration {
	multiplier := 1 << min(attempt, 30)
	delay := base * time.Duration(multiplier)

	// Dodanie lekkiego jittera przy użyciu lokalnego generatora (unikamy deprecated rand.Seed)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	jitter := time.Duration(rng.Intn(1000)) * time.Millisecond
	delay += jitter

	if delay > maxVal {
		return maxVal
	}
	return delay
}

var _ = (*RegistrationWorker)(nil)
