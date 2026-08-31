package shared

import (
	"context"
	"log/slog"
	"time"
)

// PostgresRotatable definiuje interfejs dla puli DB, która potrafi przyjąć nowe poświadczenia
type PostgresRotatable interface {
	UpdateCredentials(username string, password []byte) error
}

// StartPostgresRotationLoop uruchamia powtarzalny proces w tle dla rotacji poświadczeń DB
func StartPostgresRotationLoop(ctx context.Context, cfg Config, interval time.Duration, db PostgresRotatable) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("🛑 [SDK] Zatrzymywanie pętli rotacji sekretów DB...")
			return
		case <-ticker.C:
			slog.Info("🔄 [SDK] Odświeżanie poświadczeń PostgreSQL z agenta...")

			refreshCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			newCreds, cleanup, err := RefreshPostgres(refreshCtx, cfg)
			cancel()

			if err != nil {
				slog.Error("⚠️ [SDK] Błąd pobierania nowych poświadczeń DB", "err", err)
				continue
			}

			// Podmieniamy poświadczenia w działającej puli połączeń
			if db != nil {
				if err := db.UpdateCredentials(newCreds.Username, newCreds.Password); err != nil {
					slog.Error("❌ [SDK] Nie udało się podmienić haseł w puli DB", "err", err)
				} else {
					slog.Info("✅ [SDK] Pomyślnie zaktualizowano poświadczenia DB w puli")
				}
			}

			// Wyszyszczenie bajtów pamięci RAM po rotacji
			cleanup()
		}
	}
}
