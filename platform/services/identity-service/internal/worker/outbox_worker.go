package worker

import (
	"context"
)

// OutboxWorker odpowiada za pobieranie wiadomości z tabeli outbox i publikowanie ich do RabbitMQ.
type OutboxWorker struct {
	// TODO: Dodać zależności (outboxRepo, publisher, interval)
}

func NewOutboxWorker() *OutboxWorker {
	return &OutboxWorker{}
}

func (w *OutboxWorker) Start(ctx context.Context) {
	// TODO: Zaimplementować pętlę z time.Ticker i obsługą ctx.Done()
}
