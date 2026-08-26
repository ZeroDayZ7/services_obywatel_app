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

type MessagingHMACConfig struct {
	TargetKeys map[string]KeyTarget `mapstructure:"HMAC_TARGET_KEYS"`
}

type Config struct {
	Server   viper.ServerConfig  `mapstructure:",squash"`
	Database viper.DBConfig      `mapstructure:",squash"`
	OTEL     viper.OTELConfig    `mapstructure:",squash"`
	KMS      viper.KMSConfig     `mapstructure:",squash"`
	HMAC     MessagingHMACConfig `mapstructure:",squash"`
	Shutdown time.Duration       `mapstructure:"SHUTDOWN_TIMEOUT" validate:"required"`
}

var AppConfig Config

func (c *Config) ToKMSServiceConfig() kms.Config {
	return c.KMS.ToKMSServiceConfig()
}

func LoadConfigGlobal() error {
	log := shared.GetLogger()

	viper.SetBaseDefaults("messaging-service")
	viper.SetDBDefaults()
	viper.SetRedisDefaults()
	viper.SetKMSDefaults()

	// Nadawcy zewnętrzni (Gateway)
	spfViper.SetDefault("HMAC_TARGET_KEYS", map[string]KeyTarget{
		"gateway": {
			TargetKey: "hmac-gateway-messaging",
			Algorithm: "HmacSha256",
		},
	})

	if err := viper.InitConfig(&AppConfig, "messaging-service"); err != nil {
		return fmt.Errorf("failed to initialize messaging-service config: %w", err)
	}

	log.Info("messaging-service configuration loaded successfully")
	return nil
}
