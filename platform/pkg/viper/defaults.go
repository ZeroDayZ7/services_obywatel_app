package viper

import "github.com/spf13/viper"

func SetSharedDefaults(serviceName string) {
	// Server defaults
	viper.SetDefault("APP_NAME", "http-server")
	viper.SetDefault("PORT", "8081")
	viper.SetDefault("BODY_LIMIT_MB", 2)
	viper.SetDefault("APP_VERSION", "2.1.1")
	viper.SetDefault("ENV", "development")
	viper.SetDefault("SERVER_HEADER", "ZeroDayZ7")
	viper.SetDefault("PREFORK", false)
	viper.SetDefault("CASE_SENSITIVE", true)
	viper.SetDefault("STRICT_ROUTING", false)

	// Czas z jednostkami (s = sekundy, m = minuty, h = godziny)
	viper.SetDefault("IDLE_TIMEOUT", "30s")
	viper.SetDefault("READ_TIMEOUT", "15s")
	viper.SetDefault("WRITE_TIMEOUT", "15s")

	// Wspólne dla wszystkich mikroserwisów
	viper.SetDefault("REDIS_HOST", "127.0.0.1")
	viper.SetDefault("REDIS_PORT", "6379")
	viper.SetDefault("REDIS_PASSWORD", "")
	viper.SetDefault("REDIS_DB", 0)

	// OTEL
	viper.SetDefault("OTEL_ENABLED", true)
	viper.SetDefault("OTEL_ENDPOINT", "http://localhost:4318/v1/traces")
	viper.SetDefault("OTEL_SERVICE_NAME", serviceName)

	// JWT
	viper.SetDefault("JWT_ACCESS_SECRET", "super_access_secret_123")
	viper.SetDefault("JWT_REFRESH_SECRET", "super_refresh_secret_123")
	viper.SetDefault("JWT_ACCESS_TTL", "15m")
	viper.SetDefault("JWT_REFRESH_TTL", "168h") // 7 dni

	// Shutdown i Proxy
	viper.SetDefault("SHUTDOWN_TIMEOUT", "5s")
	viper.SetDefault("PROXY_MAX_IDLE_CONNS", 100)
	viper.SetDefault("PROXY_IDLE_CONN_TIMEOUT", "90s")
	viper.SetDefault("PROXY_MAX_IDLE_CONNS_PER_HOST", 20)
	viper.SetDefault("PROXY_REQUEST_TIMEOUT", "30s")

	// Session
	viper.SetDefault("REDIS_SESSION_PREFIX", "session:")
	viper.SetDefault("REDIS_SESSION_TTL", "60m")

	// Services URLs
	viper.SetDefault("SERVICE_AUTH_URL", "http://localhost:8082")
	viper.SetDefault("SERVICE_DOCS_URL", "http://localhost:8083")
	viper.SetDefault("SERVICE_NOTIFY_URL", "http://localhost:8084")
	viper.SetDefault("SERVICE_USERS_URL", "http://localhost:3000")

	viper.SetDefault("INTERNAL_HMAC_SECRET", "change_me_in_production")
}
