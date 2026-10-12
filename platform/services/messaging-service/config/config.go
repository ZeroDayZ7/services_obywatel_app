package config

import (
	"fmt"
	"maps"
	"time"

	spfViper "github.com/spf13/viper"
	"github.com/zerodayz7/platform/pkg/kms"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
)

type KeyTarget struct {
	TargetKey string `mapstructure:"target_key"`
	Algorithm string `mapstructure:"algorithm"`
}

type MessagingHMACConfig struct {
	TargetKeys   map[string]KeyTarget `mapstructure:"HMAC_TARGET_KEYS" validate:"required"`
	InternalKeys map[string]KeyTarget `mapstructure:"HMAC_INTERNAL_KEYS" validate:"required"`
}

type Config struct {
	Server          viper.ServerConfig  `mapstructure:",squash"`
	Database        viper.DBConfig      `mapstructure:",squash"`
	OTEL            viper.OTELConfig    `mapstructure:",squash"`
	KMS             viper.KMSConfig     `mapstructure:",squash"`
	Redis           viper.RedisConfig   `mapstructure:",squash"`
	HMAC            MessagingHMACConfig `mapstructure:",squash"`
	RabbitMQEnabled bool                `mapstructure:"RABBITMQ_ENABLED"`
	Shutdown        time.Duration       `mapstructure:"SHUTDOWN_TIMEOUT" validate:"required"`
}

var AppConfig Config

func (c *Config) ToKMSServiceConfig() kms.Config {
	return c.KMS.ToKMSServiceConfig()
}

func (c *Config) GetAllSecurityKeys() map[string]KeyTarget {
	allKeys := make(map[string]KeyTarget)
	maps.Copy(allKeys, c.HMAC.TargetKeys)
	maps.Copy(allKeys, c.HMAC.InternalKeys)
	return allKeys
}

func LoadConfigGlobal() error {
	log := shared.GetLogger()

	viper.SetBaseDefaults("messaging-service")
	viper.SetDBDefaults()
	viper.SetRedisDefaults()
	viper.SetKMSDefaults()

	spfViper.SetConfigName("config")
	spfViper.SetConfigType("yaml")
	spfViper.AddConfigPath(".")
	spfViper.AddConfigPath("./config")

	if err := spfViper.ReadInConfig(); err != nil {
		if _, ok := err.(spfViper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		log.Warn("No config.yaml file found, falling back to environment variables and defaults")
	}

	if err := viper.InitConfig(&AppConfig, "messaging-service"); err != nil {
		return fmt.Errorf("failed to initialize messaging-service config: %w", err)
	}

	log.Info("messaging-service configuration loaded successfully")
	return nil
}
