package config

import (
	"crypto/ed25519"
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

type ProxyConfig struct {
	MaxIdleConns        int           `mapstructure:"PROXY_MAX_IDLE_CONNS" validate:"min=1"`
	MaxIdleConnsPerHost int           `mapstructure:"PROXY_MAX_IDLE_CONNS_PER_HOST" validate:"min=1"`
	IdleConnTimeout     time.Duration `mapstructure:"PROXY_IDLE_CONN_TIMEOUT" validate:"required"`
	RequestTimeout      time.Duration `mapstructure:"PROXY_REQUEST_TIMEOUT" validate:"required"`
}

type JWTConfig struct {
	AccessPublicKey ed25519.PublicKey `mapstructure:"-"`
}

type GatewayHMACConfig struct {
	TargetKeys  map[string]KeyTarget `mapstructure:"HMAC_TARGET_KEYS"`
	RabbitMQKey KeyTarget            `mapstructure:"HMAC_RABBITMQ_KEY"`
}

type Config struct {
	Server           viper.ServerConfig   `mapstructure:",squash"`
	Redis            viper.RedisConfig    `mapstructure:",squash"`
	Session          viper.SessionConfig  `mapstructure:",squash"`
	HMAC             GatewayHMACConfig    `mapstructure:",squash"`
	Services         viper.ServicesConfig `mapstructure:",squash"`
	OTEL             viper.OTELConfig     `mapstructure:",squash"`
	RabbitMQ         viper.RabbitMQConfig `mapstructure:",squash"`
	KMS              viper.KMSConfig      `mapstructure:",squash"`
	Proxy            ProxyConfig          `mapstructure:",squash"`
	JWT              JWTConfig            `mapstructure:",squash"`
	CORSAllowOrigins string               `mapstructure:"CORS_ALLOW_ORIGINS" validate:"required"`
	Shutdown         time.Duration        `mapstructure:"SHUTDOWN_TIMEOUT" validate:"required"`
}

//#region ToKMSServiceConfig
func (c *Config) ToKMSServiceConfig() kms.Config {
	return c.KMS.ToKMSServiceConfig()
}

var AppConfig Config

//#region SetGatewayDefaults
func SetGatewayDefaults() {
	viper.SetBaseDefaults("gateway")
	viper.SetRedisDefaults()
	viper.SetSessionDefaults()
	viper.SetKMSDefaults()

	spfViper.SetDefault("HMAC_TARGET_KEYS", map[string]KeyTarget{
		"auth-service": {
			TargetKey: "hmac-gateway-auth",
			Algorithm: "HmacSha256",
		},
		"identity-service": {
			TargetKey: "hmac-gateway-identity",
			Algorithm: "HmacSha256",
		},
		"citizen-docs-service": {
			TargetKey: "hmac-gateway-docs",
			Algorithm: "HmacSha256",
		},
		"messaging-service": {
			TargetKey: "hmac-gateway-messaging",
			Algorithm: "HmacSha256",
		},
		"officer-bff": {
			TargetKey: "hmac-gateway-officer-bff",
			Algorithm: "HmacSha256",
		},
	})

	spfViper.SetDefault("HMAC_RABBITMQ_KEY", KeyTarget{
		TargetKey: "hmac-gateway-rabbitmq",
		Algorithm: "HmacSha256",
	})
}

//#region LoadConfigGlobal
func LoadConfigGlobal() error {
	log := shared.GetLogger()

	SetGatewayDefaults()

	if err := viper.InitConfig(&AppConfig, "gateway"); err != nil {
		return fmt.Errorf("failed to initialize gateway config: %w", err)
	}

	log.Info("Gateway configuration loaded successfully")
	return nil
}
