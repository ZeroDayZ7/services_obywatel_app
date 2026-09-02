package agent

import (
	"context"
	"time"

	"github.com/zerodayz7/platform/pkg/shared"
)

type PostgresRotatable interface {
	UpdateCredentials(username string, password []byte) error
}

type RedisRotatable interface {
	UpdateCredentials(password []byte) error
}

// StartAutoRotation inicjalizuje goroutines rotacji na podstawie wczytanego manifestu.
func StartAutoRotation(ctx context.Context, manifest *Manifest, db PostgresRotatable) error {
	log := shared.GetLogger()

	for _, spec := range manifest.Credentials {
		if !spec.Enabled || spec.RotationInterval <= 0 {
			log.InfoMap("⏸️ Rotacja wyłączona dla zasobu", map[string]any{"resource": spec.Name})
			continue
		}

		switch spec.Type {
		case "postgres":
			if db == nil {
				continue
			}
			go runPostgresRotationLoop(ctx, manifest.SocketPath, manifest.Timeout, spec, db)

		default:
			log.WarnObj("Brak obsługi rotacji dla typu zasobu", map[string]any{"type": spec.Type})
		}
	}

	return nil
}

func runPostgresRotationLoop(ctx context.Context, socketPath string, timeout time.Duration, spec ResourceSpec, db PostgresRotatable) {
	log := shared.GetLogger()

	ticker := time.NewTicker(spec.RotationInterval)
	defer ticker.Stop()

	log.InfoMap("⏱️ Uruchomiono automatyczną pętlę rotacji Postgres", map[string]any{
		"resource": spec.Name,
		"interval": spec.RotationInterval.String(),
	})

	for {
		select {
		case <-ctx.Done():
			log.InfoMap("🛑 Zatrzymano pętlę rotacji Postgres", map[string]any{"resource": spec.Name})
			return

		case <-ticker.C:
			log.InfoMap("🔄 Odświeżanie poświadczeń z Agenta...", map[string]any{"resource": spec.Name})

			fetchCtx, cancel := context.WithTimeout(ctx, timeout)
			creds, cleanup, err := FetchAgentSecret[PostgresCredentials](fetchCtx, socketPath, timeout, spec.Name)
			cancel()

			if err != nil {
				log.WarnObj("⚠️ Nie udało się pobrać nowych poświadczeń", map[string]any{
					"resource": spec.Name,
					"err":      err.Error(),
				})
				continue
			}

			if err := db.UpdateCredentials(creds.Username, creds.Password); err != nil {
				log.WarnObj("❌ Błąd aktualizacji puli DB", map[string]any{
					"resource": spec.Name,
					"err":      err.Error(),
				})
			} else {
				log.InfoMap("✅ Pomyślnie zaktualizowano poświadczenia w puli DB", map[string]any{"resource": spec.Name})
			}

			// Czyszczenie bajtów pamięci RAM po rotacji
			clear(creds.Password)
			cleanup()
		}
	}
}
