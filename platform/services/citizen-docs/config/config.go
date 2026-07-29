package config

import (
	"time"

	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
)

type SecurityConfig struct {
	DocsEncryptionKey string `mapstructure:"DOCS_ENCRYPTION_KEY" validate:"required,len=32"`
	DocsPeselSalt     string `mapstructure:"DOCS_PESEL_SALT" validate:"required,min=16"`
	HMACSecret        string `mapstructure:"INTERNAL_HMAC_SECRET" validate:"required,min=32"`
}

type Config struct {
	Server   viper.ServerConfig  `mapstructure:",squash"`
	Database viper.DBConfig      `mapstructure:",squash"`
	Redis    viper.RedisConfig   `mapstructure:",squash"`
	Session  viper.SessionConfig `mapstructure:",squash"`
	OTEL     viper.OTELConfig    `mapstructure:",squash"`
	Internal SecurityConfig      `mapstructure:",squash"`
	Shutdown time.Duration       `mapstructure:"SHUTDOWN_TIMEOUT_SEC" validate:"required"`
}

var (
	AppConfig Config
	Store     *session.Store
)

func LoadConfigGlobal() error {
	log := shared.GetLogger()

	if err := viper.InitConfig(&AppConfig, "citizen-docs"); err != nil {
		return err
	}

	log.Info("Citizen-Docs configuration loaded successfully")
	return nil
}
