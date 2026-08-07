// cmdr: viper\defaults.go

package viper

import "github.com/spf13/viper"

func SetBaseDefaults(serviceName string) {
	viper.SetDefault("APP_NAME", serviceName)
	viper.SetDefault("PORT", "8081")
	viper.SetDefault("BODY_LIMIT_MB", 2)
	viper.SetDefault("ENV", "development")
	viper.SetDefault("IDLE_TIMEOUT", "30s")
	viper.SetDefault("READ_TIMEOUT", "15s")
	viper.SetDefault("WRITE_TIMEOUT", "15s")
	viper.SetDefault("SHUTDOWN_TIMEOUT", "10s")

	viper.SetDefault("OTEL_ENABLED", false)
	viper.SetDefault("OTEL_ENDPOINT", "http://localhost:4318/v1/traces")
	viper.SetDefault("OTEL_SERVICE_NAME", serviceName)
}

func SetDBDefaults() {
	viper.SetDefault("DB_HOST", "127.0.0.1")
	viper.SetDefault("DB_PORT", 5432)
	viper.SetDefault("DB_USER", "postgres")
	viper.SetDefault("DB_PASSWORD", "secret")
	viper.SetDefault("DB_NAME", "platform")
	viper.SetDefault("DB_SSLMODE", "disable")
	viper.SetDefault("DB_MAX_OPEN_CONNS", 10)
	viper.SetDefault("DB_MAX_IDLE_CONNS", 5)
	viper.SetDefault("DB_CONN_MAX_LIFETIME", "1h")
}

func SetRedisDefaults() {
	viper.SetDefault("REDIS_HOST", "127.0.0.1")
	viper.SetDefault("REDIS_PORT", "6379")
	viper.SetDefault("REDIS_PASSWORD", "")
	viper.SetDefault("REDIS_DB", 0)
	viper.SetDefault("REDIS_POOL_SIZE", 10)
	viper.SetDefault("REDIS_MIN_IDLE_CONNS", 2)
	viper.SetDefault("REDIS_POOL_TIMEOUT", "4s")
	viper.SetDefault("REDIS_TIMEOUT", "5s")
}

func SetSessionDefaults() {
	viper.SetDefault("SESSION_TTL", "24h")
}

func SetKMSDefaults() {
	viper.SetDefault("KMS_ENDPOINT", "http://127.0.0.1:8080")
}
