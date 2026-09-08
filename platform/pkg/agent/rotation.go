package agent

import (
	"bytes"
	"context"
	"strings"
	"time"

	"github.com/zerodayz7/platform/pkg/shared"
)

type PostgresRotatable interface {
	UpdateCredentials(username string, password []byte) error
}

type RedisRotatable interface {
	UpdateCredentials(password []byte) error
}

// StartAutoRotation inicjalizuje goroutines rotacji dla bazy danych oraz opcjonalnie dla Redisa.
func StartAutoRotation(ctx context.Context, manifest *Manifest, db PostgresRotatable, rdb RedisRotatable) error {
	log := shared.GetLogger()

	for _, spec := range manifest.Credentials {
		if !spec.Enabled || spec.RotationInterval <= 0 {
			log.InfoMap("Rotacja wyłączona dla zasobu", map[string]any{"resource": spec.Name})
			continue
		}

		switch spec.Type {
		case "postgres":
			if db == nil {
				continue
			}
			go runPostgresRotationLoop(ctx, manifest.SocketPath, manifest.Timeout, spec, db)

		case "redis":
			if rdb == nil {
				continue
			}
			go runRedisRotationLoop(ctx, manifest.SocketPath, manifest.Timeout, spec, rdb)

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

	log.InfoMap("Uruchomiono automatyczną pętlę rotacji Postgres", map[string]any{
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
				log.WarnObj("Nie udało się pobrać nowych poświadczeń", map[string]any{
					"resource": spec.Name,
					"err":      err.Error(),
				})
				continue
			}

			// Oczyszczenie bajtów hasła z ewentualnych znaki nowej linii / NUL
			cleanPass := bytes.Trim(creds.Password, "\x00\r\n\t ")

			if err := db.UpdateCredentials(strings.TrimSpace(creds.Username), cleanPass); err != nil {
				log.WarnObj("❌ Błąd aktualizacji puli DB", map[string]any{
					"resource": spec.Name,
					"err":      err.Error(),
				})
			} else {
				log.InfoMap("✅ Pomyślnie zaktualizowano poświadczenia w puli DB", map[string]any{"resource": spec.Name})
			}

			clear(creds.Password)
			cleanup()
		}
	}
}

func runRedisRotationLoop(ctx context.Context, socketPath string, timeout time.Duration, spec ResourceSpec, rdb RedisRotatable) {
	log := shared.GetLogger()

	ticker := time.NewTicker(spec.RotationInterval)
	defer ticker.Stop()

	log.InfoMap("Uruchomiono automatyczną pętlę rotacji Redis", map[string]any{
		"resource": spec.Name,
		"interval": spec.RotationInterval.String(),
	})

	for {
		select {
		case <-ctx.Done():
			log.InfoMap("🛑 Zatrzymano pętlę rotacji Redis", map[string]any{"resource": spec.Name})
			return

		case <-ticker.C:
			log.InfoMap("🔄 Odświeżanie poświadczeń Redisa z Agenta...", map[string]any{"resource": spec.Name})

			fetchCtx, cancel := context.WithTimeout(ctx, timeout)
			creds, cleanup, err := FetchAgentSecret[RedisCredentials](fetchCtx, socketPath, timeout, spec.Name)
			cancel()

			if err != nil {
				log.WarnObj("Nie udało się pobrać nowych poświadczeń Redisa", map[string]any{
					"resource": spec.Name,
					"err":      err.Error(),
				})
				continue
			}

			if err := rdb.UpdateCredentials(creds.Password); err != nil {
				log.WarnObj("❌ Błąd aktualizacji klienta Redis", map[string]any{
					"resource": spec.Name,
					"err":      err.Error(),
				})
			} else {
				log.InfoMap("✅ Pomyślnie zaktualizowano poświadczenia Redisa", map[string]any{"resource": spec.Name})
			}

			clear(creds.Password)
			cleanup()
		}
	}
}
