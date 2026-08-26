package config

import (
	"fmt"
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

type DocsHMACConfig struct {
	TargetKeys map[string]KeyTarget `mapstructure:"HMAC_TARGET_KEYS"`
	PeselKey   KeyTarget            `mapstructure:"HMAC_PESEL_KEY"`
}

type SecurityConfig struct {
	DocsPeselSalt string `mapstructure:"DOCS_PESEL_SALT" validate:"required,min=16"`
}

type Config struct {
	Server   viper.ServerConfig  `mapstructure:",squash"`
	Database viper.DBConfig      `mapstructure:",squash"`
	Redis    viper.RedisConfig   `mapstructure:",squash"`
	Session  viper.SessionConfig `mapstructure:",squash"`
	OTEL     viper.OTELConfig    `mapstructure:",squash"`
	KMS      viper.KMSConfig     `mapstructure:",squash"`
	HMAC     DocsHMACConfig      `mapstructure:",squash"`
	Security SecurityConfig      `mapstructure:",squash"`
	Shutdown time.Duration       `mapstructure:"SHUTDOWN_TIMEOUT" validate:"required"`
}

var AppConfig Config

func (c *Config) ToKMSServiceConfig() kms.Config {
	return c.KMS.ToKMSServiceConfig()
}

func LoadConfigGlobal() error {
	log := shared.GetLogger()

	viper.SetBaseDefaults("citizen-docs")
	viper.SetDBDefaults()
	viper.SetRedisDefaults()
	viper.SetSessionDefaults()
	viper.SetKMSDefaults()

	// Nadawcy zewnętrzni (np. API Gateway / BFF)
	spfViper.SetDefault("HMAC_TARGET_KEYS", map[string]KeyTarget{
		"gateway": {
			TargetKey: "hmac-gateway-docs",
			Algorithm: "HmacSha256",
		},
		"officer-bff": {
			TargetKey: "hmac-bff-docs",
			Algorithm: "HmacSha256",
		},
	})

	// Wewnętrzny klucz domenowy do szyfrowania/indeksowania PESEL
	spfViper.SetDefault("HMAC_PESEL_KEY", KeyTarget{
		TargetKey: "hmac-docs-pesel-index",
		Algorithm: "HmacSha256",
	})

	if err := viper.InitConfig(&AppConfig, "citizen-docs"); err != nil {
		return fmt.Errorf("failed to initialize citizen-docs config: %w", err)
	}

	log.Info("Citizen-Docs configuration loaded successfully")
	return nil
}
