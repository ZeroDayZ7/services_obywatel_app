package config

import (
	"fmt"

	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
)

// AppConfig przechowuje globalną konfigurację serwisu (używamy wspólnego typu z pkg)
var AppConfig viper.Config

// LoadConfigGlobal wczytuje konfigurację przy użyciu ujednoliconego mechanizmu platformy
func LoadConfigGlobal() error {
	log := shared.GetLogger()

	// Wywołujemy uniwersalną funkcję z pkg, która:
	// 1. Ustawi defaulty dla serwisu notification-service
	// 2. Wczyta plik .env
	// 3. Zmapuje wartości na strukturę (obsługując time.Duration typu "30s")
	// 4. Przeprowadzi walidację (czy podano port, sekrety itp.)
	if err := viper.InitConfig(&AppConfig, "notification-service"); err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	log.Info("Configuration loaded and validated for notification-service")
	return nil
}
