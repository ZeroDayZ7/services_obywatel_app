package config

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
)

type Config struct {
	Server   viper.ServerConfig           `mapstructure:",squash"`
	Database viper.DBConfig               `mapstructure:",squash"`
	OTEL     viper.OTELConfig             `mapstructure:",squash"`
	KMS      viper.KMSConfig              `mapstructure:",squash"`
	Internal viper.InternalSecurityConfig `mapstructure:",squash"`
	Shutdown time.Duration                `mapstructure:"SHUTDOWN_TIMEOUT" validate:"required"`
}

var (
	AppConfig Config
	Store     *session.Store
)

func LoadConfigGlobal() error {
	log := shared.GetLogger()

	viper.SetDBDefaults()
	viper.SetRedisDefaults()
	viper.SetKMSDefaults()

	if err := viper.InitConfig(&AppConfig, "messaging-service"); err != nil {
		return fmt.Errorf("failed to initialize messaging-service config: %w", err)
	}

	log.Info("messaging-service configuration loaded successfully")
	return nil
}
