package shared

import (
	"context"
	"log/slog"
	"time"
)

// PostgresRotatable definiuje interfejs dla puli DB, która potrafi przyjąć nowe hasło/DSN
type PostgresRotatable interface {
	UpdateCredentials(username, password string) error
}

// StartPostgresRotationLoop uruchamia powtarzalny proces w tle dla dowolnego serwisu
func (c *Client) StartPostgresRotationLoop(ctx context.Context, interval time.Duration, db PostgresRotatable) {
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
			newCreds, err := c.RefreshPostgres(refreshCtx)
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

			// Natychmiast zerujemy nowe hasło ze zmiennej w pamięci RAM
			newCreds.Password = ""
		}
	}
}
