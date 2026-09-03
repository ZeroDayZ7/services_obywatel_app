package config

import (
	"maps"
	"time"

	"github.com/zerodayz7/platform/pkg/kms"
	"github.com/zerodayz7/platform/pkg/viper"
	"github.com/zerodayz7/services/identity-service/internal/worker"
)

type KeyTarget struct {
	TargetKey string `mapstructure:"target_key"`
	Algorithm string `mapstructure:"algorithm"`
}

type IdentityHMACConfig struct {
	TargetKeys   map[string]KeyTarget `mapstructure:"HMAC_TARGET_KEYS"`
	InternalKeys map[string]KeyTarget `mapstructure:"HMAC_INTERNAL_KEYS"`
}

type RabbitMQConsumersConfig struct {
	TrustedSenders map[string]KeyTarget `mapstructure:"RABBITMQ_TRUSTED_SENDERS"`
}

type PDFConfig struct {
	ChromeWSURL string `mapstructure:"CHROME_WS_URL" validate:"required"`
}

type AuditWorkerConfig struct {
	Enabled       bool          `mapstructure:"AUDIT_WORKER_ENABLED"`
	BatchSize     int           `mapstructure:"AUDIT_WORKER_BATCH_SIZE" validate:"min=1"`
	Interval      time.Duration `mapstructure:"AUDIT_WORKER_INTERVAL"`
	MaxRetries    int           `mapstructure:"AUDIT_WORKER_MAX_RETRIES" validate:"min=0"`
	BackoffBase   time.Duration `mapstructure:"AUDIT_WORKER_BACKOFF_BASE"`
	BackoffMax    time.Duration `mapstructure:"AUDIT_WORKER_BACKOFF_MAX"`
	Concurrency   int           `mapstructure:"AUDIT_WORKER_CONCURRENCY" validate:"min=1"`
	RoutingKey    string        `mapstructure:"AUDIT_WORKER_ROUTING_KEY"`
	SourceService string        `mapstructure:"AUDIT_WORKER_SOURCE_SERVICE"`
}

type RegistrationWorkerConfig struct {
	Enabled     bool          `mapstructure:"REGISTRATION_WORKER_ENABLED"`
	BatchSize   int           `mapstructure:"REGISTRATION_WORKER_BATCH_SIZE" validate:"min=1"`
	Interval    time.Duration `mapstructure:"REGISTRATION_WORKER_INTERVAL"`
	MaxRetries  int           `mapstructure:"REGISTRATION_WORKER_MAX_RETRIES" validate:"min=0"`
	BackoffBase time.Duration `mapstructure:"REGISTRATION_WORKER_BACKOFF_BASE"`
	BackoffMax  time.Duration `mapstructure:"REGISTRATION_WORKER_BACKOFF_MAX"`
	Concurrency int           `mapstructure:"REGISTRATION_WORKER_CONCURRENCY" validate:"min=1"`
	RoutingKey  string        `mapstructure:"REGISTRATION_WORKER_ROUTING_KEY"`
}

type Config struct {
	Server             viper.ServerConfig       `mapstructure:",squash"`
	Database           viper.DBConfig           `mapstructure:",squash"`
	Redis              viper.RedisConfig        `mapstructure:",squash"`
	RabbitMQ           viper.RabbitMQConfig     `mapstructure:",squash"`
	S3                 viper.S3Config           `mapstructure:",squash"`
	HMAC               IdentityHMACConfig       `mapstructure:",squash"`
	RabbitConsumers    RabbitMQConsumersConfig  `mapstructure:",squash"`
	KMS                viper.KMSConfig          `mapstructure:",squash"`
	OTEL               viper.OTELConfig         `mapstructure:",squash"`
	PDF                PDFConfig                `mapstructure:",squash"`
	AuditWorker        AuditWorkerConfig        `mapstructure:",squash"`
	RegistrationWorker RegistrationWorkerConfig `mapstructure:",squash"`
	Shutdown           time.Duration            `mapstructure:"SHUTDOWN_TIMEOUT" validate:"required"`
}

//#region ToKMSServiceConfig
func (c *Config) ToKMSServiceConfig() kms.Config {
	return c.KMS.ToKMSServiceConfig()
}

func (c *AuditWorkerConfig) ToWorkerConfig() worker.AuditWorkerConfig {
	return worker.AuditWorkerConfig{
		BatchSize:     c.BatchSize,
		Interval:      c.Interval,
		MaxRetries:    c.MaxRetries,
		BackoffBase:   c.BackoffBase,
		BackoffMax:    c.BackoffMax,
		Concurrency:   c.Concurrency,
		RoutingKey:    c.RoutingKey,
		SourceService: c.SourceService,
	}
}

func (c *RegistrationWorkerConfig) ToWorkerConfig() worker.RegistrationWorkerConfig {
	return worker.RegistrationWorkerConfig{
		BatchSize:   c.BatchSize,
		Interval:    c.Interval,
		MaxRetries:  c.MaxRetries,
		BackoffBase: c.BackoffBase,
		BackoffMax:  c.BackoffMax,
		Concurrency: c.Concurrency,
		RoutingKey:  c.RoutingKey,
	}
}

//#region GetAllSecurityKeys
func (c *Config) GetAllSecurityKeys() map[string]KeyTarget {
	allKeys := make(map[string]KeyTarget)

	// 1. Zewnętrzni nadawcy (API Gateway, BFF)
	maps.Copy(allKeys, c.HMAC.TargetKeys)

	// 2. Zaufani nadawcy RabbitMQ
	maps.Copy(allKeys, c.RabbitConsumers.TrustedSenders)

	// 3. Klucze wewnętrzne
	maps.Copy(allKeys, c.HMAC.InternalKeys)

	return allKeys
}
