package viper

import "time"

type ServicesConfig struct {
	Auth      string `mapstructure:"SERVICE_AUTH_URL" validate:"required,url"`
	Documents string `mapstructure:"SERVICE_DOCS_URL" validate:"required,url"`
	Notify    string `mapstructure:"SERVICE_NOTIFY_URL" validate:"required,url"`
	Users     string `mapstructure:"SERVICE_USERS_URL" validate:"required,url"`
}

type InternalSecurityConfig struct {
	// HMAC musi mieć co najmniej 32 znaki dla realnego bezpieczeństwa (używany do podpisów)
	HMACSecret string `mapstructure:"INTERNAL_HMAC_SECRET" validate:"required,min=32"`

	// EncryptionKey musi mieć dokładnie 32 znaki dla AES-256 (szyfrowanie danych PII)
	EncryptionKey string `mapstructure:"INTERNAL_ENCRYPTION_KEY" validate:"required,len=32"`

	// HashSalt używany do "solenia" hashy PESEL/Email (zapobiega rainbow tables)
	HashSalt string `mapstructure:"INTERNAL_HASH_SALT" validate:"omitempty,min=16"`
}

type OTELConfig struct {
	Enabled     bool   `mapstructure:"OTEL_ENABLED"`
	Endpoint    string `mapstructure:"OTEL_ENDPOINT" validate:"required_if=Enabled true"`
	ServiceName string `mapstructure:"OTEL_SERVICE_NAME" validate:"required_if=Enabled true"`
}

type DBConfig struct {
	// omitempty sprawia, że jeśli nie ma DSN (np. w Gateway), walidacja przejdzie
	DSN             string        `mapstructure:"DATABASE_DSN" validate:"omitempty"`
	MaxOpenConns    int           `mapstructure:"DB_MAX_OPEN_CONNS" validate:"omitempty,min=1"`
	MaxIdleConns    int           `mapstructure:"DB_MAX_IDLE_CONNS" validate:"omitempty,min=1"`
	ConnMaxLifetime time.Duration `mapstructure:"DB_CONN_MAX_LIFETIME"`
}

type ProxyConfig struct {
	MaxIdleConns        int           `mapstructure:"PROXY_MAX_IDLE_CONNS" validate:"min=1"`
	IdleConnTimeout     time.Duration `mapstructure:"PROXY_IDLE_CONN_TIMEOUT"`
	MaxIdleConnsPerHost int           `mapstructure:"PROXY_MAX_IDLE_CONNS_PER_HOST" validate:"min=1"`
	RequestTimeout      time.Duration `mapstructure:"PROXY_REQUEST_TIMEOUT"`
}

type SessionConfig struct {
	TTL time.Duration `mapstructure:"REDIS_SESSION_TTL" validate:"required"`
}

type ServerConfig struct {
	AppName       string        `mapstructure:"APP_NAME" validate:"required"`
	Port          string        `mapstructure:"PORT" validate:"required,numeric"`
	BodyLimitMB   int           `mapstructure:"BODY_LIMIT_MB" validate:"min=1"`
	Env           string        `mapstructure:"ENV" validate:"required,oneof=development staging production"`
	AppVersion    string        `mapstructure:"APP_VERSION"`
	ServerHeader  string        `mapstructure:"SERVER_HEADER"`
	Prefork       bool          `mapstructure:"PREFORK"`
	CaseSensitive bool          `mapstructure:"CASE_SENSITIVE"`
	StrictRouting bool          `mapstructure:"STRICT_ROUTING"`
	IdleTimeout   time.Duration `mapstructure:"IDLE_TIMEOUT"`
	ReadTimeout   time.Duration `mapstructure:"READ_TIMEOUT"`
	WriteTimeout  time.Duration `mapstructure:"WRITE_TIMEOUT"`
}

type RedisConfig struct {
	Host     string `mapstructure:"REDIS_HOST" validate:"required"`
	Port     string `mapstructure:"REDIS_PORT" validate:"required,numeric"`
	Password string `mapstructure:"REDIS_PASSWORD"`
	DB       int    `mapstructure:"REDIS_DB" validate:"min=0"`
}

type JWTConfig struct {
	AccessSecret  string        `mapstructure:"JWT_ACCESS_SECRET" validate:"required,min=16"`
	RefreshSecret string        `mapstructure:"JWT_REFRESH_SECRET" validate:"required,min=16"`
	AccessTTL     time.Duration `mapstructure:"JWT_ACCESS_TTL" validate:"required"`
	RefreshTTL    time.Duration `mapstructure:"JWT_REFRESH_TTL" validate:"required"`
}

type Config struct {
	Server    ServerConfig           `mapstructure:",squash"`
	Redis     RedisConfig            `mapstructure:",squash"`
	Session   SessionConfig          `mapstructure:",squash"`
	Proxy     ProxyConfig            `mapstructure:",squash"`
	CORSAllow string                 `mapstructure:"CORS_ALLOW_ORIGINS" validate:"required"`
	Shutdown  time.Duration          `mapstructure:"SHUTDOWN_TIMEOUT" validate:"required"`
	JWT       JWTConfig              `mapstructure:",squash"`
	OTEL      OTELConfig             `mapstructure:",squash"`
	Internal  InternalSecurityConfig `mapstructure:",squash"`
	Services  ServicesConfig         `mapstructure:",squash"`
	Database  DBConfig               `mapstructure:",squash"`
}
