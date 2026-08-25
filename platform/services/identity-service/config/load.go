package config

import (
	"fmt"

	spfViper "github.com/spf13/viper"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/viper"
)

var AppConfig Config

// LoadConfigGlobal loads configuration into AppConfig and applies defaults.
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
