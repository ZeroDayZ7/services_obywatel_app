package config

import (
	"fmt"
	"time"

	spfViper "github.com/spf13/viper"
	"github.com/zerodayz7/platform/pkg/kms"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
)

type BFFHMACConfig struct {
	TargetKeys map[string]string `mapstructure:"HMAC_TARGET_KEYS"`
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

func (c *Config) ToKMSServiceConfig() kms.Config {
	return c.KMS.ToKMSServiceConfig()
}

var AppConfig Config

func SetBFFDefaults() {
	viper.SetBaseDefaults("officer_bff")
	viper.SetRedisDefaults()
	viper.SetKMSDefaults()
	viper.SetGatewayHMACDefaults()

	spfViper.SetDefault("AUTH_ACCESS_TOKEN_TTL", "15m")
	spfViper.SetDefault("AUTH_REFRESH_TOKEN_TTL", "168h")

	// Definiujemy, które klucze z KMS są potrzebne dla BFF
	// 1. Klucz do weryfikacji przychodzącego ruchu z Gatewaya
	// 2. Klucze do podpisywania wychodzącego ruchu do mikroserwisów
	spfViper.SetDefault("HMAC_TARGET_KEYS", map[string]string{
		"gateway-service":      "hmac-gateway-officer-bff", // do odbioru ruchu z GW
		"auth-service":         "hmac-bff-auth",            // do wysyłania do Auth
		"identity-service":     "hmac-bff-identity",        // do wysyłania do Identity
		"citizen-docs-service": "hmac-bff-docs",            // do wysyłania do Docs
	})
}

func LoadConfigGlobal() error {
	log := shared.GetLogger()

	SetBFFDefaults()

	if err := viper.InitConfig(&AppConfig, "officer_bff"); err != nil {
		return fmt.Errorf("failed to initialize officer_bff config: %w", err)
	}

	log.Info("Officer_bff configuration loaded successfully")
	return nil
}
