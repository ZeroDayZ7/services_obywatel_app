// cmdr: viper\types.go

package viper

import (
	"fmt"
	"strings"
	"time"
)

// #region
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

// #region
type DBConfig struct {
	Host            string        `mapstructure:"DB_HOST" validate:"required"`
	Port            int           `mapstructure:"DB_PORT" validate:"required"`
	User            string        `mapstructure:"DB_USER" validate:"required"`
	Password        string        `mapstructure:"DB_PASSWORD" validate:"required"`
	DBName          string        `mapstructure:"DB_NAME" validate:"required"`
	SSLMode         string        `mapstructure:"DB_SSLMODE" validate:"required"`
	MaxOpenConns    int           `mapstructure:"DB_MAX_OPEN_CONNS" validate:"omitempty,min=1"`
	MaxIdleConns    int           `mapstructure:"DB_MAX_IDLE_CONNS" validate:"omitempty,min=1"`
	ConnMaxLifetime time.Duration `mapstructure:"DB_CONN_MAX_LIFETIME"`
}

// #region
type RedisConfig struct {
	Host         string        `mapstructure:"REDIS_HOST" validate:"required"`
	Port         string        `mapstructure:"REDIS_PORT" validate:"required,numeric"`
	Password     string        `mapstructure:"REDIS_PASSWORD"`
	DB           int           `mapstructure:"REDIS_DB" validate:"min=0"`
	PoolSize     int           `mapstructure:"REDIS_POOL_SIZE" validate:"min=1"`
	MinIdleConns int           `mapstructure:"REDIS_MIN_IDLE_CONNS" validate:"min=0"`
	PoolTimeout  time.Duration `mapstructure:"REDIS_POOL_TIMEOUT" validate:"required"`
	Timeout      time.Duration `mapstructure:"REDIS_TIMEOUT" validate:"required"`
}

// #region
type InternalSecurityConfig struct {
	HMACSecret string `mapstructure:"-"`
}

// #region
type OTELConfig struct {
	Enabled     bool   `mapstructure:"OTEL_ENABLED"`
	Endpoint    string `mapstructure:"OTEL_ENDPOINT" validate:"required_if=Enabled true"`
	ServiceName string `mapstructure:"OTEL_SERVICE_NAME" validate:"required_if=Enabled true"`
}

// #region
type RabbitMQConfig struct {
	Enabled  bool   `mapstructure:"RABBITMQ_ENABLED"`
	Host     string `mapstructure:"RABBITMQ_HOST" validate:"required_if=Enabled true"`
	Port     int    `mapstructure:"RABBITMQ_PORT" validate:"required_if=Enabled true"`
	User     string `mapstructure:"RABBITMQ_USER" validate:"required_if=Enabled true"`
	Password string `mapstructure:"RABBITMQ_PASSWORD" validate:"required_if=Enabled true"`
	VHost    string `mapstructure:"RABBITMQ_VHOST"`
}

// #region
type KMSConfig struct {
	Endpoint      string `mapstructure:"KMS_ENDPOINT" validate:"required,url"`
	ServiceSecret string `mapstructure:"KMS_SERVICE_SECRET" validate:"required,min=32"`
}

// #region
type SessionConfig struct {
	TTL time.Duration `mapstructure:"SESSION_TTL" validate:"required"`
}

// #region
type ServicesConfig struct {
	Auth      string `mapstructure:"SERVICE_AUTH_URL" validate:"required,url"`
	Documents string `mapstructure:"SERVICE_DOCS_URL" validate:"required,url"`
	Notify    string `mapstructure:"SERVICE_NOTIFY_URL" validate:"required,url"`
	Messaging string `mapstructure:"SERVICE_MESSAGING_URL" validate:"required,url"`
	WS        string `mapstructure:"SERVICE_WS_URL" validate:"required"`
}

// #region GetURL
func (r RabbitMQConfig) GetURL() string {
	vhost := r.VHost
	if vhost == "" {
		vhost = "/"
	}
	if !strings.HasPrefix(vhost, "/") {
		vhost = "/" + vhost
	}
	return fmt.Sprintf("amqp://%s:%s@%s:%d%s", r.User, r.Password, r.Host, r.Port, vhost)
}

// #region GetDSN
func (cfg DBConfig) GetDSN() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		cfg.Host, cfg.User, cfg.Password, cfg.DBName, cfg.Port, cfg.SSLMode,
	)
}
