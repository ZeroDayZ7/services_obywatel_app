package config

import (
	"fmt"
	"time"

	spfViper "github.com/spf13/viper"
	"github.com/zerodayz7/platform/pkg/kms"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
)

type IdentityHMACConfig struct {
	TargetKeys map[string]string `mapstructure:"HMAC_TARGET_KEYS"`
	AuditKey   string            `mapstructure:"HMAC_AUDIT_KEY"`
}

type RabbitMQConsumersConfig struct {
	TrustedSenders map[string]string `mapstructure:"RABBITMQ_TRUSTED_SENDERS"`
}

type Config struct {
	Server          viper.ServerConfig      `mapstructure:",squash"`
	Database        viper.DBConfig          `mapstructure:",squash"`
	Redis           viper.RedisConfig       `mapstructure:",squash"`
	RabbitMQ        viper.RabbitMQConfig    `mapstructure:",squash"`
	HMAC            IdentityHMACConfig      `mapstructure:",squash"`
	RabbitConsumers RabbitMQConsumersConfig `mapstructure:",squash"`
	KMS             viper.KMSConfig         `mapstructure:",squash"`
	OTEL            viper.OTELConfig        `mapstructure:",squash"`
	Shutdown        time.Duration           `mapstructure:"SHUTDOWN_TIMEOUT" validate:"required"`
}

func (c *Config) ToKMSServiceConfig() kms.Config {
	return c.KMS.ToKMSServiceConfig(c.Server.AppName)
}

var AppConfig Config

func LoadConfigGlobal() error {
	log := shared.GetLogger()

	viper.SetDBDefaults()
	viper.SetRedisDefaults()
	viper.SetKMSDefaults()

	// Domyślne mapowanie nadawców ruchu HTTP na nazwy kluczy w KMS
	spfViper.SetDefault("HMAC_TARGET_KEYS", map[string]string{
		"gateway":     "hmac-gateway-identity",
		"officer-bff": "hmac-bff-identity",
	})

	// Domyślne mapowanie zaufanych nadawców zdarzeń z RabbitMQ (jeśli istnieją)
	spfViper.SetDefault("RABBITMQ_TRUSTED_SENDERS", map[string]string{
		"auth-service": "hmac-auth-rabbitmq",
	})

	spfViper.SetDefault("HMAC_AUDIT_KEY", "identity-audit-hmac")

	if err := viper.InitConfig(&AppConfig, "identity_service"); err != nil {
		return fmt.Errorf("failed to initialize identity_service config: %w", err)
	}

	log.Info("Identity_service configuration loaded successfully")
	return nil
}
