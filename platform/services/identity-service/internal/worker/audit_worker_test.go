package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeAuditRepo struct {
	pending       []AuditEvent
	markSyncedIDs []uuid.UUID
	markRetryIDs  []uuid.UUID
	failFetch     error
}

func (f *fakeAuditRepo) FetchPending(ctx context.Context, limit int, leaseWindow time.Duration) ([]AuditEvent, error) {
	if f.failFetch != nil {
		return nil, f.failFetch
	}
	out := make([]AuditEvent, len(f.pending))
	copy(out, f.pending)
	f.pending = nil
	return out, nil
}

func (f *fakeAuditRepo) MarkSynced(ctx context.Context, ids []uuid.UUID) error {
	f.markSyncedIDs = append(f.markSyncedIDs, ids...)
	return nil
}

func (f *fakeAuditRepo) MarkRetry(ctx context.Context, id uuid.UUID, retryCount int, lastErr string) error {
	f.markRetryIDs = append(f.markRetryIDs, id)
	return nil
}

func (f *fakeAuditRepo) MarkFailed(ctx context.Context, id uuid.UUID, retryCount int, lastErr string) error {
	return nil
}

func (f *fakeAuditRepo) PendingStats(ctx context.Context) (int, time.Duration, error) {
	return 0, 0, nil
}

type fakeAuditPublisher struct {
	publishErr error
	calls      int
}

func (f *fakeAuditPublisher) Publish(ctx context.Context, routingKey string, payload []byte) error {
	f.calls++
	return f.publishErr
}

func (f *fakeAuditPublisher) Close() error { return nil }

func TestAuditWorker_EmptyQueue(t *testing.T) {
	repo := &fakeAuditRepo{}
	pub := &fakeAuditPublisher{}
	worker := NewAuditWorkerWithDeps(repo, pub, AuditWorkerConfig{BatchSize: 10, Interval: time.Second, MaxRetries: 3, BackoffBase: time.Second, BackoffMax: 5 * time.Second, Concurrency: 1, RoutingKey: "audit.log.created", SourceService: "identity-service"})

	if err := worker.processOnce(context.Background()); err != nil {
		t.Fatalf("processOnce returned error for empty queue: %v", err)
	}
	if pub.calls != 0 {
		t.Fatalf("expected no publish calls for empty queue, got %d", pub.calls)
	}
}

func TestAuditWorker_ProcessesBatch(t *testing.T) {
	eventID := uuid.New()
	repo := &fakeAuditRepo{pending: []AuditEvent{{ID: eventID, Action: "CITIZEN_REGISTERED", UserID: uuid.New(), ActorID: uuid.New(), CreatedAt: time.Now()}}}
	pub := &fakeAuditPublisher{}
	worker := NewAuditWorkerWithDeps(repo, pub, AuditWorkerConfig{BatchSize: 10, Interval: time.Second, MaxRetries: 3, BackoffBase: time.Second, BackoffMax: 5 * time.Second, Concurrency: 1, RoutingKey: "audit.log.created", SourceService: "identity-service"})

	if err := worker.processOnce(context.Background()); err != nil {
		t.Fatalf("processOnce returned error: %v", err)
	}
	if pub.calls != 1 {
		t.Fatalf("expected 1 publish call, got %d", pub.calls)
	}
	if len(repo.markSyncedIDs) != 1 || repo.markSyncedIDs[0] != eventID {
		t.Fatalf("expected event to be marked synced, got %#v", repo.markSyncedIDs)
	}
}

func TestAuditWorker_RetryableFailure(t *testing.T) {
	eventID := uuid.New()
	repo := &fakeAuditRepo{pending: []AuditEvent{{ID: eventID, Action: "CITIZEN_REGISTERED", UserID: uuid.New(), ActorID: uuid.New(), CreatedAt: time.Now()}}}
	pub := &fakeAuditPublisher{publishErr: errors.New("temporary network issue")}
	worker := NewAuditWorkerWithDeps(repo, pub, AuditWorkerConfig{BatchSize: 10, Interval: time.Second, MaxRetries: 1, BackoffBase: time.Second, BackoffMax: 5 * time.Second, Concurrency: 1, RoutingKey: "audit.log.created", SourceService: "identity-service"})

	if err := worker.processOnce(context.Background()); err != nil {
		t.Fatalf("expected retryable failure handled, got: %v", err)
	}
	if len(repo.markRetryIDs) != 1 || repo.markRetryIDs[0] != eventID {
		t.Fatalf("expected retry mark for failed event, got %#v", repo.markRetryIDs)
	}
}
