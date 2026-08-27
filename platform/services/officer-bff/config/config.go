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

type BFFHMACConfig struct {
	TargetKeys  map[string]KeyTarget `mapstructure:"HMAC_TARGET_KEYS"`
	RabbitMQKey KeyTarget            `mapstructure:"HMAC_RABBITMQ_KEY"`
}

type Config struct {
	Server                viper.ServerConfig   `mapstructure:",squash"`
	Redis                 viper.RedisConfig    `mapstructure:",squash"`
	RabbitMQ              viper.RabbitMQConfig `mapstructure:",squash"`
	HMAC                  BFFHMACConfig        `mapstructure:",squash"`
	KMS                   viper.KMSConfig      `mapstructure:",squash"`
	OTEL                  viper.OTELConfig     `mapstructure:",squash"`
	Shutdown              time.Duration        `mapstructure:"SHUTDOWN_TIMEOUT" validate:"required"`
	AuthServiceURL        string               `mapstructure:"AUTH_SERVICE_URL" validate:"required"`
	IdentityServiceURL    string               `mapstructure:"IDENTITY_SERVICE_URL" validate:"required"`
	CitizenDocsServiceURL string               `mapstructure:"CITIZEN_DOCS_SERVICE_URL" validate:"required"`

	AccessTokenTTL  time.Duration `mapstructure:"AUTH_ACCESS_TOKEN_TTL" validate:"required"`
	RefreshTokenTTL time.Duration `mapstructure:"AUTH_REFRESH_TOKEN_TTL" validate:"required"`
}

//#region ToKMSServiceConfig
func (c *Config) ToKMSServiceConfig() kms.Config {
	return c.KMS.ToKMSServiceConfig()
}

func SetBFFDefaults() {
	viper.SetBaseDefaults("officer_bff")
	viper.SetRedisDefaults()
	viper.SetKMSDefaults()

	spfViper.SetDefault("AUTH_ACCESS_TOKEN_TTL", "15m")
	spfViper.SetDefault("AUTH_REFRESH_TOKEN_TTL", "168h")

	spfViper.SetDefault("HMAC_TARGET_KEYS", map[string]KeyTarget{
		"gateway-service": {
			TargetKey: "hmac-gateway-officer-bff",
			Algorithm: "HmacSha256",
		},
		"auth-service": {
			TargetKey: "hmac-bff-auth",
			Algorithm: "HmacSha256",
		},
		"identity-service": {
			TargetKey: "hmac-bff-identity",
			Algorithm: "HmacSha256",
		},
		"citizen-docs-service": {
			TargetKey: "hmac-bff-docs",
			Algorithm: "HmacSha256",
		},
	})

	spfViper.SetDefault("HMAC_RABBITMQ_KEY", KeyTarget{
		TargetKey: "hmac-officer-bff-rabbitmq",
		Algorithm: "HmacSha256",
	})
}

//#region LoadConfigGlobal
func Load() (*Config, error) {
	log := shared.GetLogger()

	SetBFFDefaults()

	var cfg Config
	if err := viper.InitConfig(&cfg, "officer_bff"); err != nil {
		return nil, fmt.Errorf("failed to initialize officer_bff config: %w", err)
	}

	log.Info("Officer_bff configuration loaded successfully")
	return &cfg, nil
}
