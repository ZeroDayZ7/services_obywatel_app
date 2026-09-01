// cmdr: viper\defaults.go

package viper

import "github.com/spf13/viper"

//#region SetBaseDefaults
func SetBaseDefaults(serviceName string) {
	viper.SetDefault("APP_NAME", serviceName)
	viper.SetDefault("PORT", "8081")
	viper.SetDefault("BODY_LIMIT_MB", 2)
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("IDLE_TIMEOUT", "30s")
	viper.SetDefault("READ_TIMEOUT", "15s")
	viper.SetDefault("WRITE_TIMEOUT", "15s")
	viper.SetDefault("SHUTDOWN_TIMEOUT", "10s")

	viper.SetDefault("OTEL_ENABLED", false)
	viper.SetDefault("OTEL_ENDPOINT", "http://localhost:4318/v1/traces")
	viper.SetDefault("OTEL_SERVICE_NAME", serviceName)
}

//#region SetDBDefaults
func SetDBDefaults() {
	viper.SetDefault("DB_HOST", "xxxx")
	viper.SetDefault("DB_PORT", 5432)
	viper.SetDefault("DB_USER", "postgres")
	viper.SetDefault("DB_PASSWORD", "secret")
	viper.SetDefault("DB_NAME", "platform")
	viper.SetDefault("DB_SSLMODE", "disable")
	viper.SetDefault("DB_MAX_OPEN_CONNS", 10)
	viper.SetDefault("DB_MAX_IDLE_CONNS", 5)
	viper.SetDefault("DB_CONN_MAX_LIFETIME", "1h")
}

//#region SetRedisDefaults
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

//#region SetSessionDefaults
func SetSessionDefaults() {
	viper.SetDefault("SESSION_TTL", "24h")
}

//#region SetKMSDefaults
func SetKMSDefaults() {
	viper.SetDefault("KMS_ENDPOINT", "http://127.0.0.1:8080")
	viper.SetDefault("KMS_TIMEOUT", "2s")
}

//#region SetHMACDefaults
func SetHMACDefaults() {
	viper.SetDefault("HMAC_HEADER_NAME", "X-HMAC-Signature")
}

//#region SetGatewayHMACDefaults
func SetGatewayHMACDefaults() {
	SetHMACDefaults()
	viper.SetDefault("HMAC_TARGET_KEYS", map[string]string{})
}

//#region SetS3Defaults
func SetS3Defaults() {
	viper.SetDefault("S3_ENABLED", false)
	viper.SetDefault("S3_ENDPOINT", "localhost:9000")
	viper.SetDefault("S3_ACCESS_KEY", "minioadmin")
	viper.SetDefault("S3_SECRET_KEY", "minioadminpassword")
	viper.SetDefault("S3_BUCKET", "citizens-data")
	viper.SetDefault("S3_REGION", "us-east-1")
	viper.SetDefault("S3_USE_SSL", false)
}
