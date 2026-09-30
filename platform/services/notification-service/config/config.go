package config

import (
	"fmt"
	"time"

	spfViper "github.com/spf13/viper"
	"github.com/zerodayz7/platform/pkg/kms"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
)

type NotificationHMACConfig struct {
	TargetKeys map[string]KeyTarget `mapstructure:"HMAC_TARGET_KEYS"`
}

type KeyTarget struct {
	TargetKey string `mapstructure:"target_key"`
	Algorithm string `mapstructure:"algorithm"`
}

type Config struct {
	Server   viper.ServerConfig           `mapstructure:",squash"`
	Database viper.DBConfig               `mapstructure:",squash"`
	Redis    viper.RedisConfig            `mapstructure:",squash"`
	RabbitMQ viper.RabbitMQConfig         `mapstructure:",squash"`
	KMS      viper.KMSConfig              `mapstructure:",squash"`
	HMAC     NotificationHMACConfig       `mapstructure:",squash"`
	Internal viper.InternalSecurityConfig `mapstructure:",squash"`
	OTEL     viper.OTELConfig             `mapstructure:",squash"`
	Shutdown time.Duration                `mapstructure:"SHUTDOWN_TIMEOUT" validate:"required"`
}

var AppConfig Config

func (c *Config) ToKMSServiceConfig() kms.Config {
	return c.KMS.ToKMSServiceConfig()
}

func LoadConfigGlobal() error {
	log := shared.GetLogger()

	viper.SetBaseDefaults("notification-service")
	viper.SetDBDefaults()
	viper.SetRedisDefaults()
	viper.SetKMSDefaults()
	spfViper.SetDefault("HMAC_TARGET_KEYS", map[string]KeyTarget{
		"gateway": {
			TargetKey: "hmac-gateway-notification",
			Algorithm: "HmacSha256",
		},
	})

	if err := viper.InitConfig(&AppConfig, "notification-service"); err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	log.Info("Configuration loaded and validated for notification-service")
	return nil
}
