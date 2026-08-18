package config

import (
	"fmt"
	"time"

	"github.com/zerodayz7/platform/pkg/kms"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
)

type Config struct {
	Server   viper.ServerConfig           `mapstructure:",squash"`
	Database viper.DBConfig               `mapstructure:",squash"`
	Redis    viper.RedisConfig            `mapstructure:",squash"`
	RabbitMQ viper.RabbitMQConfig         `mapstructure:",squash"`
	Internal viper.InternalSecurityConfig `mapstructure:",squash"`
	KMS      viper.KMSConfig              `mapstructure:",squash"`
	OTEL     viper.OTELConfig             `mapstructure:",squash"`
	Shutdown time.Duration                `mapstructure:"SHUTDOWN_TIMEOUT" validate:"required"`
}

func (c *Config) ToKMSServiceConfig() kms.Config {
	return kms.Config{
		Endpoint:      c.KMS.Endpoint,
		ServiceName:   c.Server.AppName,
		ServiceSecret: c.KMS.ServiceSecret,
	}
}

var AppConfig Config

func LoadConfigGlobal() error {
	log := shared.GetLogger()

	viper.SetDBDefaults()
	viper.SetRedisDefaults()
	viper.SetKMSDefaults()

	if err := viper.InitConfig(&AppConfig, "identity_service"); err != nil {
		return fmt.Errorf("failed to initialize identity_service config: %w", err)
	}

	log.Info("Identity_service configuration loaded successfully")
	return nil
}
