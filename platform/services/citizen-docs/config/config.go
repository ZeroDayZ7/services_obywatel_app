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

type HMACConfig struct {
	TargetKeys   map[string]KeyTarget `mapstructure:"HMAC_TARGET_KEYS"`
	InternalKeys map[string]KeyTarget `mapstructure:"HMAC_INTERNAL_KEYS"`
}

type Config struct {
	Server   viper.ServerConfig  `mapstructure:",squash"`
	Database viper.DBConfig      `mapstructure:",squash"`
	Redis    viper.RedisConfig   `mapstructure:",squash"`
	Session  viper.SessionConfig `mapstructure:",squash"`
	OTEL     viper.OTELConfig    `mapstructure:",squash"`
	KMS      viper.KMSConfig     `mapstructure:",squash"`
	HMAC     HMACConfig          `mapstructure:",squash"`
	Shutdown time.Duration       `mapstructure:"SHUTDOWN_TIMEOUT" validate:"required"`
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

	viper.SetBaseDefaults("citizen-docs")
	viper.SetDBDefaults()
	viper.SetRedisDefaults()
	viper.SetSessionDefaults()
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

	if err := viper.InitConfig(&AppConfig, "citizen-docs"); err != nil {
		return fmt.Errorf("failed to initialize citizen-docs config: %w", err)
	}

	log.Info("Citizen-Docs configuration loaded successfully")
	return nil
}
