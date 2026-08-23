package config

import (
	"crypto/ed25519"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2/middleware/session"
	spfViper "github.com/spf13/viper"
	"github.com/zerodayz7/platform/pkg/kms"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
)

type JWTConfig struct {
	SigningMode     string            `mapstructure:"JWT_SIGNING_MODE"`
	KeyID           string            `mapstructure:"JWT_KEY_ID"`
	AccessTTL       time.Duration     `mapstructure:"JWT_ACCESS_TTL" validate:"required"`
	RefreshTTL      time.Duration     `mapstructure:"JWT_REFRESH_TTL" validate:"required"`
	AccessPublicKey ed25519.PublicKey `mapstructure:"-"`
}

type AuthHMACConfig struct {
	TargetKeys map[string]string `mapstructure:"HMAC_TARGET_KEYS"`
}

type RabbitMQConsumersConfig struct {
	TrustedSenders map[string]string `mapstructure:"RABBITMQ_TRUSTED_SENDERS"`
}

type Config struct {
	Server          viper.ServerConfig      `mapstructure:",squash"`
	Database        viper.DBConfig          `mapstructure:",squash"`
	Redis           viper.RedisConfig       `mapstructure:",squash"`
	Session         viper.SessionConfig     `mapstructure:",squash"`
	HMAC            AuthHMACConfig          `mapstructure:",squash"`
	OTEL            viper.OTELConfig        `mapstructure:",squash"`
	RabbitMQ        viper.RabbitMQConfig    `mapstructure:",squash"`
	RabbitConsumers RabbitMQConsumersConfig `mapstructure:",squash"`
	KMS             viper.KMSConfig         `mapstructure:",squash"`
	JWT             JWTConfig               `mapstructure:",squash"`
	Shutdown        time.Duration           `mapstructure:"SHUTDOWN_TIMEOUT" validate:"required"`
}

func (c *Config) ToKMSServiceConfig() kms.Config {
	return c.KMS.ToKMSServiceConfig()
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

	spfViper.SetDefault("HMAC_TARGET_KEYS", map[string]string{
		"gateway":     "hmac-gateway-auth",
		"officer-bff": "hmac-bff-auth",
	})

	spfViper.SetDefault("RABBITMQ_TRUSTED_SENDERS", map[string]string{
		"identity-service": "hmac-identity-rabbitmq",
	})

	if err := viper.InitConfig(&AppConfig, "auth-service"); err != nil {
		return fmt.Errorf("failed to initialize auth-service config: %w", err)
	}

	log.Info("Auth-service configuration loaded successfully")
	return nil
}
