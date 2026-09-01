package config

import (
	"crypto/ed25519"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2/middleware/session"
	spfViper "github.com/spf13/viper"
	"github.com/zerodayz7/platform/pkg/kms"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
)

type AuthConfig struct {
	Domain string `mapstructure:"AUTH_DOMAIN" validate:"required"`
}

type AgentConfig struct {
	SocketPath string `mapstructure:"SECRET_AGENT_SOCKET_PATH"`
}

type KeyTarget struct {
	TargetKey string `mapstructure:"target_key"`
	Algorithm string `mapstructure:"algorithm"`
}

type JWTConfig struct {
	SigningMode     string            `mapstructure:"JWT_SIGNING_MODE"`
	KeyID           string            `mapstructure:"JWT_KEY_ID"`
	AccessTTL       time.Duration     `mapstructure:"JWT_ACCESS_TTL" validate:"required"`
	RefreshTTL      time.Duration     `mapstructure:"JWT_REFRESH_TTL" validate:"required"`
	AccessPublicKey ed25519.PublicKey `mapstructure:"-"`
}

type AuthHMACConfig struct {
	TargetKeys  map[string]KeyTarget `mapstructure:"HMAC_TARGET_KEYS"`
	RabbitMQKey KeyTarget            `mapstructure:"HMAC_RABBITMQ_KEY"`
}

type RabbitMQConsumersConfig struct {
	TrustedSenders map[string]KeyTarget `mapstructure:"RABBITMQ_TRUSTED_SENDERS"`
}

type Config struct {
	Server          viper.ServerConfig      `mapstructure:",squash"`
	Database        viper.DBConfig          `mapstructure:",squash"`
	Redis           viper.RedisConfig       `mapstructure:",squash"`
	Session         viper.SessionConfig     `mapstructure:",squash"`
	Auth            AuthConfig              `mapstructure:",squash"`
	HMAC            AuthHMACConfig          `mapstructure:",squash"`
	OTEL            viper.OTELConfig        `mapstructure:",squash"`
	RabbitMQ        viper.RabbitMQConfig    `mapstructure:",squash"`
	RabbitConsumers RabbitMQConsumersConfig `mapstructure:",squash"`
	KMS             viper.KMSConfig         `mapstructure:",squash"`
	JWT             JWTConfig               `mapstructure:",squash"`
	Agent           AgentConfig             `mapstructure:",squash"`
	Shutdown        time.Duration           `mapstructure:"SHUTDOWN_TIMEOUT" validate:"required"`
}

//#region ToKMSServiceConfig
func (c *Config) ToKMSServiceConfig() kms.Config {
	return c.KMS.ToKMSServiceConfig()
}

func (c *Config) IsLocalDev() bool {
	return c.Server.Env == "local" || c.Server.Env == "development" || c.Server.Env == "dev"
}

var (
	AppConfig Config
	Store     *session.Store
)

//#region LoadConfigGlobal
func LoadConfigGlobal() error {
	log := shared.GetLogger()

	viper.SetDBDefaults()
	viper.SetRedisDefaults()
	viper.SetKMSDefaults()

	// W LoadConfigGlobal() dopisz na początku:
	spfViper.SetDefault("SECRET_AGENT_SOCKET_PATH", "/var/run/agent-sockets/agent.sock")

	spfViper.SetDefault("HMAC_TARGET_KEYS", map[string]KeyTarget{
		"gateway": {
			TargetKey: "hmac-gateway-auth",
			Algorithm: "HmacSha256",
		},
		"officer-bff": {
			TargetKey: "hmac-bff-auth",
			Algorithm: "HmacSha256",
		},
	})

	spfViper.SetDefault("RABBITMQ_TRUSTED_SENDERS", map[string]KeyTarget{
		"identity-service": {
			TargetKey: "hmac-identity-rabbitmq",
			Algorithm: "HmacSha256",
		},
	})

	spfViper.SetDefault("HMAC_RABBITMQ_KEY", KeyTarget{
		TargetKey: "hmac-auth-rabbitmq",
		Algorithm: "HmacSha256",
	})

	spfViper.SetDefault("AUTH_DOMAIN", "obywatel.gov.pl")

	if err := viper.InitConfig(&AppConfig, "auth-service"); err != nil {
		return fmt.Errorf("failed to initialize auth-service config: %w", err)
	}

	log.Info("Auth-service configuration loaded successfully")
	return nil
}
