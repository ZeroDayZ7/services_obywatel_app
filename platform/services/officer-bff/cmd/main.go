package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/services/officer-bff/config"
	"github.com/zerodayz7/services/officer-bff/internal/di"
	"github.com/zerodayz7/services/officer-bff/internal/router"
	"github.com/zerodayz7/services/officer-bff/internal/security"
)

//#region main
func main() {
	log := shared.GetLogger()

	// 1. Inicjalizacja konfiguracji
	cfg, err := config.Load()
	if err != nil {
		log.Error("❌ Nie udało się załadować konfiguracji", "error", err)
		os.Exit(1)
	}

	// 2. Konfiguracja kontekstu Graceful Shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 3. Inicjalizacja magazynu kluczy ze wspólnego pakietu httpserver
	keyStore := httpserver.NewKeyStore()

	// 4. Załadowanie kluczy bezpieczeństwa z KMS z timeoutem 10s
	securityCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := security.LoadSecurityKeys(securityCtx, cfg, keyStore); err != nil {
		log.Error("❌ Błąd ładowania kluczy bezpieczeństwa", "error", err)
		os.Exit(1)
	}

	// 5. Budowanie kontenera DI
	container := di.BuildContainer(cfg, keyStore)

	// 6. Budowanie routera HTTP
	r := router.NewRouter(container)

	// 7. Konfiguracja serwera HTTP
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// 8. Graceful Shutdown
	if err := httpserver.Run(srv, cfg.Shutdown); err != nil {
		log.Error("Server forced shutdown with error", "error", err)
	}
}
