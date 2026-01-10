package viper

import "time"

type ServicesConfig struct {
	Auth      string `mapstructure:"SERVICE_AUTH_URL"`
	Documents string `mapstructure:"SERVICE_DOCS_URL"`
	Notify    string `mapstructure:"SERVICE_NOTIFY_URL"`
	Users     string `mapstructure:"SERVICE_USERS_URL"`
}

type InternalSecurityConfig struct {
	HMACSecret string `mapstructure:"INTERNAL_HMAC_SECRET"`
}

type OTELConfig struct {
	Enabled     bool   `mapstructure:"OTEL_ENABLED"`
	Endpoint    string `mapstructure:"OTEL_ENDPOINT"`
	ServiceName string `mapstructure:"OTEL_SERVICE_NAME"`
}

type DBConfig struct {
	DSN             string        `mapstructure:"DATABASE_DSN"`
	MaxOpenConns    int           `mapstructure:"DB_MAX_OPEN_CONNS"`
	MaxIdleConns    int           `mapstructure:"DB_MAX_IDLE_CONNS"`
	ConnMaxLifetime time.Duration `mapstructure:"DB_CONN_MAX_LIFETIME"`
}

type ProxyConfig struct {
	MaxIdleConns        int           `mapstructure:"PROXY_MAX_IDLE_CONNS"`
	IdleConnTimeout     time.Duration `mapstructure:"PROXY_IDLE_CONN_TIMEOUT"`
	MaxIdleConnsPerHost int           `mapstructure:"PROXY_MAX_IDLE_CONNS_PER_HOST"`
	RequestTimeout      time.Duration `mapstructure:"PROXY_REQUEST_TIMEOUT"`
}

type SessionConfig struct {
	Prefix string        `mapstructure:"REDIS_SESSION_PREFIX"`
	TTL    time.Duration `mapstructure:"REDIS_SESSION_TTL"`
}

type ServerConfig struct {
	AppName       string        `mapstructure:"APP_NAME"`
	Port          string        `mapstructure:"PORT"`
	BodyLimitMB   int           `mapstructure:"BODY_LIMIT_MB"`
	Env           string        `mapstructure:"ENV"`
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
	Host     string `mapstructure:"REDIS_HOST"`
	Port     string `mapstructure:"REDIS_PORT"`
	Password string `mapstructure:"REDIS_PASSWORD"`
	DB       int    `mapstructure:"REDIS_DB"`
}

type JWTConfig struct {
	AccessSecret  string        `mapstructure:"JWT_ACCESS_SECRET"`
	RefreshSecret string        `mapstructure:"JWT_REFRESH_SECRET"`
	AccessTTL     time.Duration `mapstructure:"JWT_ACCESS_TTL"`
	RefreshTTL    time.Duration `mapstructure:"JWT_REFRESH_TTL"`
}

type Config struct {
	Server    ServerConfig           `mapstructure:",squash"`
	Redis     RedisConfig            `mapstructure:",squash"`
	Session   SessionConfig          `mapstructure:",squash"`
	Proxy     ProxyConfig            `mapstructure:",squash"`
	CORSAllow string                 `mapstructure:"CORS_ALLOW_ORIGINS"`
	Shutdown  time.Duration          `mapstructure:"SHUTDOWN_TIMEOUT"`
	JWT       JWTConfig              `mapstructure:",squash"`
	OTEL      OTELConfig             `mapstructure:",squash"`
	Internal  InternalSecurityConfig `mapstructure:",squash"`
	Services  ServicesConfig         `mapstructure:",squash"`
	Database  DBConfig               `mapstructure:",squash"`
}
