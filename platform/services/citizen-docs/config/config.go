package config

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
)

type SecurityConfig struct {
	DocsPeselSalt string `mapstructure:"DOCS_PESEL_SALT" validate:"required,min=16"`
}

type Config struct {
	Server   viper.ServerConfig           `mapstructure:",squash"`
	Database viper.DBConfig               `mapstructure:",squash"`
	Redis    viper.RedisConfig            `mapstructure:",squash"`
	Session  viper.SessionConfig          `mapstructure:",squash"`
	OTEL     viper.OTELConfig             `mapstructure:",squash"`
	KMS      viper.KMSConfig              `mapstructure:",squash"`
	Internal viper.InternalSecurityConfig `mapstructure:",squash"`
	Security SecurityConfig               `mapstructure:",squash"`
	Shutdown time.Duration                `mapstructure:"SHUTDOWN_TIMEOUT" validate:"required"`
}

var (
	AppConfig Config
	Store     *session.Store
)

func LoadConfigGlobal() error {
	log := shared.GetLogger()

	viper.SetBaseDefaults("citizen-docs")
	viper.SetDBDefaults()
	viper.SetRedisDefaults()
	viper.SetSessionDefaults()

	if err := viper.InitConfig(&AppConfig, "citizen-docs"); err != nil {
		return fmt.Errorf("failed to initialize citizen-docs config: %w", err)
	}

	log.Info("Citizen-Docs configuration loaded successfully")
	return nil
}
