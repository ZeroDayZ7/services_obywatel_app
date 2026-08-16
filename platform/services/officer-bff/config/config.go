package config

import (
	"fmt"
	"time"

	"github.com/zerodayz7/platform/pkg/kms"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
)

type Config struct {
	Server                viper.ServerConfig           `mapstructure:",squash"`
	Redis                 viper.RedisConfig            `mapstructure:",squash"`
	RabbitMQ              viper.RabbitMQConfig         `mapstructure:",squash"`
	Internal              viper.InternalSecurityConfig `mapstructure:",squash"`
	KMS                   viper.KMSConfig              `mapstructure:",squash"`
	OTEL                  viper.OTELConfig             `mapstructure:",squash"`
	Shutdown              time.Duration                `mapstructure:"SHUTDOWN_TIMEOUT" validate:"required"`
	AuthServiceURL        string                       `mapstructure:"AUTH_SERVICE_URL" validate:"required"`
	IdentityServiceURL    string                       `mapstructure:"IDENTITY_SERVICE_URL" validate:"required"`
	CitizenDocsServiceURL string                       `mapstructure:"CITIZEN_DOCS_SERVICE_URL" validate:"required"`
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

	viper.SetRedisDefaults()
	viper.SetKMSDefaults()

	if err := viper.InitConfig(&AppConfig, "officer_bff"); err != nil {
		return fmt.Errorf("failed to initialize officer_bff config: %w", err)
	}

	log.Info("Officer_bff configuration loaded successfully")
	return nil
}
