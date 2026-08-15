package config

import (
	"crypto/ed25519"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/zerodayz7/platform/pkg/kms"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
)

type JWTConfig struct {
	SigningMode      string             `mapstructure:"JWT_SIGNING_MODE"`
	KeyID            string             `mapstructure:"JWT_KEY_ID"`
	AccessTTL        time.Duration      `mapstructure:"JWT_ACCESS_TTL" validate:"required"`
	RefreshTTL       time.Duration      `mapstructure:"JWT_REFRESH_TTL" validate:"required"`
	AccessPrivateKey ed25519.PrivateKey `mapstructure:"-"`
}

type Config struct {
	Server   viper.ServerConfig           `mapstructure:",squash"`
	Database viper.DBConfig               `mapstructure:",squash"`
	Redis    viper.RedisConfig            `mapstructure:",squash"`
	Session  viper.SessionConfig          `mapstructure:",squash"`
	Internal viper.InternalSecurityConfig `mapstructure:",squash"`
	OTEL     viper.OTELConfig             `mapstructure:",squash"`
	RabbitMQ viper.RabbitMQConfig         `mapstructure:",squash"`
	KMS      viper.KMSConfig              `mapstructure:",squash"`
	JWT      JWTConfig                    `mapstructure:",squash"`
	Shutdown time.Duration                `mapstructure:"SHUTDOWN_TIMEOUT" validate:"required"`
}

// ToSDKConfig zwraca konfigurację KMS dla komunikacji miedzy-serwisowej
func (c *Config) ToKMSServiceConfig() kms.Config {
	return kms.Config{
		Endpoint:      c.KMS.Endpoint,
		ServiceName:   c.Server.AppName,
		ServiceSecret: c.KMS.ServiceSecret,
	}
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

	if err := viper.InitConfig(&AppConfig, "auth-service"); err != nil {
		return fmt.Errorf("failed to initialize auth-service config: %w", err)
	}

	log.Info("Auth-service configuration loaded successfully")
	return nil
}
