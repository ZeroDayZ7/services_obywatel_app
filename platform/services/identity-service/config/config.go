package config

import (
	"fmt"
	"maps"
	"time"

	spfViper "github.com/spf13/viper"
	"github.com/zerodayz7/platform/pkg/kms"
	"github.com/zerodayz7/platform/pkg/shared"
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

var AppConfig Config

//#region LoadConfigGlobal
func LoadConfigGlobal() error {
	log := shared.GetLogger()

	viper.SetDBDefaults()
	viper.SetRedisDefaults()
	viper.SetKMSDefaults()
	viper.SetS3Defaults()

	// Zewnętrzni nadawcy HTTP
	spfViper.SetDefault("HMAC_TARGET_KEYS", map[string]KeyTarget{
		"gateway": {
			TargetKey: "hmac-gateway-identity",
			Algorithm: "HmacSha256",
		},
		"officer-bff": {
			TargetKey: "hmac-bff-identity",
			Algorithm: "HmacSha256",
		},
	})

	// Zaufani nadawcy zdarzeń RabbitMQ
	spfViper.SetDefault("RABBITMQ_TRUSTED_SENDERS", map[string]KeyTarget{
		"auth-service": {
			TargetKey: "hmac-auth-rabbitmq",
			Algorithm: "HmacSha256",
		},
	})

	// Wszystkie wewnętrzne klucze serwisu zgromadzone w pojedynczym słowniku
	spfViper.SetDefault("HMAC_INTERNAL_KEYS", map[string]KeyTarget{
		"pesel": {
			TargetKey: "hmac-identity-pesel-index",
			Algorithm: "HmacSha256",
		},
		"phone": {
			TargetKey: "hmac-identity-phone-index",
			Algorithm: "HmacSha256",
		},
		"email": {
			TargetKey: "hmac-identity-email-index",
			Algorithm: "HmacSha256",
		},
		"puk": {
			TargetKey: "hmac-identity-puk-index",
			Algorithm: "HmacSha256",
		},
		"rabbitmq": {
			TargetKey: "hmac-identity-rabbitmq",
			Algorithm: "HmacSha256",
		},
		"audit": {
			TargetKey: "hmac-identity-audit",
			Algorithm: "HmacSha256",
		},
		"agreements": {
			TargetKey: "identity-agreements-key",
			Algorithm: "AES256GCM",
		},
	})

	// Audit worker defaults
	spfViper.SetDefault("AUDIT_WORKER_ENABLED", true)
	spfViper.SetDefault("AUDIT_WORKER_BATCH_SIZE", 200)
	spfViper.SetDefault("AUDIT_WORKER_INTERVAL", "2s")
	spfViper.SetDefault("AUDIT_WORKER_MAX_RETRIES", 10)
	spfViper.SetDefault("AUDIT_WORKER_BACKOFF_BASE", "1s")
	spfViper.SetDefault("AUDIT_WORKER_BACKOFF_MAX", "60s")
	spfViper.SetDefault("AUDIT_WORKER_CONCURRENCY", 1)
	spfViper.SetDefault("AUDIT_WORKER_ROUTING_KEY", "audit.log.created")
	spfViper.SetDefault("AUDIT_WORKER_SOURCE_SERVICE", "identity-service")

	// Registration worker defaults
	spfViper.SetDefault("REGISTRATION_WORKER_ENABLED", true)
	spfViper.SetDefault("REGISTRATION_WORKER_BATCH_SIZE", 100)
	spfViper.SetDefault("REGISTRATION_WORKER_INTERVAL", "2s")
	spfViper.SetDefault("REGISTRATION_WORKER_MAX_RETRIES", 5)
	spfViper.SetDefault("REGISTRATION_WORKER_BACKOFF_BASE", "1s")
	spfViper.SetDefault("REGISTRATION_WORKER_BACKOFF_MAX", "30s")
	spfViper.SetDefault("REGISTRATION_WORKER_CONCURRENCY", 2)
	spfViper.SetDefault("REGISTRATION_WORKER_ROUTING_KEY", "auth.register")

	if err := viper.InitConfig(&AppConfig, "identity-service"); err != nil {
		return fmt.Errorf("failed to initialize identity-service config: %w", err)
	}

	log.Info("Identity-service configuration loaded successfully")
	return nil
}
