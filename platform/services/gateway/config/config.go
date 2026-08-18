package config

import (
	"crypto/ed25519"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2/middleware/session"
	spfViper "github.com/spf13/viper" // import standardowego vipera do nadpisania mapy
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
)

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
	HeaderName string            `mapstructure:"HMAC_HEADER_NAME" validate:"required"`
	TargetKeys map[string]string `mapstructure:"HMAC_TARGET_KEYS"`
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

var (
	AppConfig Config
	Store     *session.Store
)

func SetGatewayDefaults() {
	viper.SetBaseDefaults("gateway")
	viper.SetRedisDefaults()
	viper.SetSessionDefaults()
	viper.SetGatewayHMACDefaults()

	// Nadpisujemy HMAC_TARGET_KEYS pełną, lokalną mapą dla Gatewaya
	spfViper.SetDefault("HMAC_TARGET_KEYS", map[string]string{
		"auth-service":         "hmac-gateway-auth",
		"identity-service":     "hmac-gateway-identity",
		"citizen-docs-service": "hmac-gateway-docs",
		"messaging-service":    "hmac-gateway-messaging",
		"officer-bff":          "hmac-gateway-officer-bff",
	})
}

func LoadConfigGlobal() error {
	log := shared.GetLogger()

	SetGatewayDefaults()

	if err := viper.InitConfig(&AppConfig, "gateway"); err != nil {
		return fmt.Errorf("failed to initialize gateway config: %w", err)
	}

	log.Info("Gateway configuration loaded successfully")
	return nil
}
