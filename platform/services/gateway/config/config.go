package config

import (
	"fmt"

	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
	pkgConfig "github.com/zerodayz7/platform/pkg/viper"
)

var (
	AppConfig viper.Config
	Store     *session.Store
)

func LoadConfigGlobal() error {
	log := shared.GetLogger()

	if err := pkgConfig.InitConfig(&AppConfig, "gateway"); err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	// 2. Walidacja (tylko to, czego automat nie sprawdzi)
	if AppConfig.Internal.HMACSecret == "" {
		return fmt.Errorf("INTERNAL_HMAC_SECRET is required")
	}

	log.Info("Configuration loaded successfully")
	return nil
}