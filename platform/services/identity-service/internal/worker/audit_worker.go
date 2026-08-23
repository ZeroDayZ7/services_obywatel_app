package worker

import (
	"context"
)

// AuditWorker odpowiada za wysyłanie logów audytowych do centralnego systemu.
type AuditWorker struct {
	// TODO: Dodać zależności (auditRepo, publisher / logger)
}

func NewAuditWorker() *AuditWorker {
	return &AuditWorker{}
}

func (w *AuditWorker) Start(ctx context.Context) {
	// TODO: Zaimplementować logikę eksportu logów audytowych
}
