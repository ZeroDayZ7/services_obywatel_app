package config

import (
	"fmt"
	"time"

	spfViper "github.com/spf13/viper"
	"github.com/zerodayz7/platform/pkg/kms"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
)

type IdentityHMACConfig struct {
	TargetKeys map[string]string `mapstructure:"HMAC_TARGET_KEYS"`
	AuditKey   string            `mapstructure:"HMAC_AUDIT_KEY"`
}

type RabbitMQConsumersConfig struct {
	TrustedSenders map[string]string `mapstructure:"RABBITMQ_TRUSTED_SENDERS"`
}

type AuditWorkerConfig struct {
	BatchSize     int           `mapstructure:"AUDIT_WORKER_BATCH_SIZE" validate:"min=1"`
	Interval      time.Duration `mapstructure:"AUDIT_WORKER_INTERVAL"`
	MaxRetries    int           `mapstructure:"AUDIT_WORKER_MAX_RETRIES" validate:"min=0"`
	BackoffBase   time.Duration `mapstructure:"AUDIT_WORKER_BACKOFF_BASE"`
	BackoffMax    time.Duration `mapstructure:"AUDIT_WORKER_BACKOFF_MAX"`
	Concurrency   int           `mapstructure:"AUDIT_WORKER_CONCURRENCY" validate:"min=1"`
	RoutingKey    string        `mapstructure:"AUDIT_WORKER_ROUTING_KEY"`
	SourceService string        `mapstructure:"AUDIT_WORKER_SOURCE_SERVICE"`
}

type Config struct {
	Server          viper.ServerConfig      `mapstructure:",squash"`
	Database        viper.DBConfig          `mapstructure:",squash"`
	Redis           viper.RedisConfig       `mapstructure:",squash"`
	RabbitMQ        viper.RabbitMQConfig    `mapstructure:",squash"`
	S3              viper.S3Config          `mapstructure:",squash"`
	HMAC            IdentityHMACConfig      `mapstructure:",squash"`
	RabbitConsumers RabbitMQConsumersConfig `mapstructure:",squash"`
	KMS             viper.KMSConfig         `mapstructure:",squash"`
	OTEL            viper.OTELConfig        `mapstructure:",squash"`
	AuditWorker     AuditWorkerConfig       `mapstructure:",squash"`
	Shutdown        time.Duration           `mapstructure:"SHUTDOWN_TIMEOUT" validate:"required"`
}

func (c *Config) ToKMSServiceConfig() kms.Config {
	return c.KMS.ToKMSServiceConfig(c.Server.AppName)
}

var AppConfig Config

func LoadConfigGlobal() error {
	log := shared.GetLogger()

	viper.SetDBDefaults()
	viper.SetRedisDefaults()
	viper.SetKMSDefaults()
	viper.SetS3Defaults()

	// Domyślne mapowanie nadawców ruchu HTTP na nazwy kluczy w KMS
	spfViper.SetDefault("HMAC_TARGET_KEYS", map[string]string{
		"gateway":     "hmac-gateway-identity",
		"officer-bff": "hmac-bff-identity",
	})

	// Domyślne mapowanie zaufanych nadawców zdarzeń z RabbitMQ (jeśli istnieją)
	spfViper.SetDefault("RABBITMQ_TRUSTED_SENDERS", map[string]string{
		"auth-service": "hmac-auth-rabbitmq",
	})

	spfViper.SetDefault("HMAC_AUDIT_KEY", "identity-audit-hmac")
	spfViper.SetDefault("AUDIT_WORKER_BATCH_SIZE", 200)
	spfViper.SetDefault("AUDIT_WORKER_INTERVAL", "2s")
	spfViper.SetDefault("AUDIT_WORKER_MAX_RETRIES", 10)
	spfViper.SetDefault("AUDIT_WORKER_BACKOFF_BASE", "1s")
	spfViper.SetDefault("AUDIT_WORKER_BACKOFF_MAX", "60s")
	spfViper.SetDefault("AUDIT_WORKER_CONCURRENCY", 1)
	spfViper.SetDefault("AUDIT_WORKER_ROUTING_KEY", "audit.log.created")
	spfViper.SetDefault("AUDIT_WORKER_SOURCE_SERVICE", "identity-service")

	spfViper.SetDefault("AGREEMENTS_KMS_KEY", "identity-agreements-key")

	if err := viper.InitConfig(&AppConfig, "identity_service"); err != nil {
		return fmt.Errorf("failed to initialize identity_service config: %w", err)
	}

	log.Info("Identity_service configuration loaded successfully")
	return nil
}
