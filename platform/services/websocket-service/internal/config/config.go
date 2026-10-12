package config

import (
	"fmt"
	"time"

	spfViper "github.com/spf13/viper"
	"github.com/zerodayz7/platform/pkg/kms"
	"github.com/zerodayz7/platform/pkg/shared"
	pkgViper "github.com/zerodayz7/platform/pkg/viper"
)

type KeyTarget struct {
	TargetKey string `mapstructure:"target_key"`
	Algorithm string `mapstructure:"algorithm"`
}

type WebSocketHMACConfig struct {
	TargetKeys map[string]KeyTarget `mapstructure:"HMAC_TARGET_KEYS"`
}

type Config struct {
	Server   pkgViper.ServerConfig `mapstructure:",squash"`
	Redis    pkgViper.RedisConfig  `mapstructure:",squash"`
	KMS      pkgViper.KMSConfig    `mapstructure:",squash"`
	HMAC     WebSocketHMACConfig   `mapstructure:",squash"`
	Shutdown time.Duration         `mapstructure:"SHUTDOWN_TIMEOUT" validate:"required"`
}

var AppConfig Config

func (c *Config) ToKMSServiceConfig() kms.Config {
	return c.KMS.ToKMSServiceConfig()
}

func LoadConfigGlobal() error {
	log := shared.GetLogger()

	pkgViper.SetBaseDefaults("websocket-service")
	pkgViper.SetRedisDefaults()
	pkgViper.SetKMSDefaults()

	spfViper.SetConfigName("config")
	spfViper.SetConfigType("yaml")
	spfViper.AddConfigPath(".")

	spfViper.SetDefault("HMAC_TARGET_KEYS", map[string]KeyTarget{
		"gateway": {
			TargetKey: "hmac-gateway-websocket",
			Algorithm: "HmacSha256",
		},
	})

	if err := spfViper.ReadInConfig(); err != nil {
		if _, ok := err.(spfViper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		log.Warn("No config.yaml file found, falling back to environment variables and defaults")
	}

	if err := pkgViper.InitConfig(&AppConfig, "websocket-service"); err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	log.Info("Configuration loaded and validated for websocket-service")
	return nil
}
