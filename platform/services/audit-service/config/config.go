package config

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/spf13/viper"
	"github.com/zerodayz7/platform/pkg/shared"
)

type ServerConfig struct {
	AppName       string
	Port          string
	BodyLimitMB   int
	Env           string
	AppVersion    string
	ServerHeader  string
	Prefork       bool
	CaseSensitive bool
	StrictRouting bool
	IdleTimeout   time.Duration
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type DBConfig struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type RateLimitConfig struct {
	Max    int
	Window time.Duration
}

type Config struct {
	Server     ServerConfig
	Redis      RedisConfig
	Database   DBConfig
	RateLimit  RateLimitConfig
	CORSAllow  string
	Shutdown   time.Duration
	SessionTTL time.Duration
}

var AppConfig Config
var Store *session.Store

func LoadConfigGlobal() error {
	log := shared.GetLogger()

	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	viper.SetDefault("APP_NAME", "http-server")
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("APP_VERSION", "0.1.0")
	viper.SetDefault("ENV", "development")
	viper.SetDefault("SERVER_HEADER", "ZeroDayZ7")
	viper.SetDefault("PREFORK", false)
	viper.SetDefault("CASE_SENSITIVE", true)
	viper.SetDefault("STRICT_ROUTING", true)
	viper.SetDefault("IDLE_TIMEOUT_SEC", 30)
	viper.SetDefault("READ_TIMEOUT_SEC", 15)
	viper.SetDefault("WRITE_TIMEOUT_SEC", 15)
	viper.SetDefault("DB_MAX_OPEN_CONNS", 50)
	viper.SetDefault("DB_MAX_IDLE_CONNS", 10)
	viper.SetDefault("DB_CONN_MAX_LIFETIME_MIN", 30)

	// Redis defaults
	viper.SetDefault("REDIS_HOST", "127.0.0.1")
	viper.SetDefault("REDIS_PORT", "6379")
	viper.SetDefault("REDIS_PASSWORD", "")
	viper.SetDefault("REDIS_DB", 0)

	// Rate limiting
	viper.SetDefault("RATE_LIMIT_MAX", 100)
	viper.SetDefault("RATE_LIMIT_WINDOW_SEC", 60)

	// Shutdown
	viper.SetDefault("SHUTDOWN_TIMEOUT_SEC", 5)

	// Session TTL (dla Redis sesji)
	viper.SetDefault("SessionTTL_MIN", 15)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.ErrorObj("Error loading .env", err)
			return fmt.Errorf("error loading .env: %v", err)
		}
	}

	AppConfig = Config{
		Server: ServerConfig{
			AppName:       viper.GetString("APP_NAME"),
			Port:          viper.GetString("PORT"),
			AppVersion:    viper.GetString("APP_VERSION"),
			Env:           viper.GetString("ENV"),
			ServerHeader:  viper.GetString("SERVER_HEADER"),
			Prefork:       viper.GetBool("PREFORK"),
			CaseSensitive: viper.GetBool("CASE_SENSITIVE"),
			StrictRouting: viper.GetBool("STRICT_ROUTING"),
			IdleTimeout:   time.Duration(viper.GetInt("IDLE_TIMEOUT_SEC")) * time.Second,
			ReadTimeout:   time.Duration(viper.GetInt("READ_TIMEOUT_SEC")) * time.Second,
			WriteTimeout:  time.Duration(viper.GetInt("WRITE_TIMEOUT_SEC")) * time.Second,
		},
		Redis: RedisConfig{
			Host:     viper.GetString("REDIS_HOST"),
			Port:     viper.GetString("REDIS_PORT"),
			Password: viper.GetString("REDIS_PASSWORD"),
			DB:       viper.GetInt("REDIS_DB"),
		},
		Database: DBConfig{
			DSN:             viper.GetString("DATABASE_DSN"),
			MaxOpenConns:    viper.GetInt("DB_MAX_OPEN_CONNS"),
			MaxIdleConns:    viper.GetInt("DB_MAX_IDLE_CONNS"),
			ConnMaxLifetime: time.Duration(viper.GetInt("DB_CONN_MAX_LIFETIME_MIN")) * time.Minute,
		},
		RateLimit: RateLimitConfig{
			Max:    viper.GetInt("RATE_LIMIT_MAX"),
			Window: time.Duration(viper.GetInt("RATE_LIMIT_WINDOW_SEC")) * time.Second,
		},
		Shutdown:   time.Duration(viper.GetInt("SHUTDOWN_TIMEOUT_SEC")) * time.Second,
		SessionTTL: time.Duration(viper.GetInt("SessionTTL_MIN")) * time.Minute,
	}

	log.Info("Auth-Service - Configuration loaded")
	return nil
}
