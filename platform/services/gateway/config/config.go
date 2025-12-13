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

type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

type Config struct {
	Server    ServerConfig
	Redis     RedisConfig
	CORSAllow string
	Shutdown  time.Duration
	JWT       JWTConfig
}

var AppConfig Config
var Store *session.Store

func LoadConfigGlobal() error {
	log := shared.GetLogger()

	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

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
	viper.SetDefault("IDLE_TIMEOUT_SEC", 30)
	viper.SetDefault("READ_TIMEOUT_SEC", 15)
	viper.SetDefault("WRITE_TIMEOUT_SEC", 15)

	// Redis defaults
	viper.SetDefault("REDIS_HOST", "127.0.0.1")
	viper.SetDefault("REDIS_PORT", "6379")
	viper.SetDefault("REDIS_PASSWORD", "")
	viper.SetDefault("REDIS_DB", 0)

	// Shutdown
	viper.SetDefault("SHUTDOWN_TIMEOUT_SEC", 5)

	// JWT
	viper.SetDefault("JWT_ACCESS_SECRET", "super_access_secret_123")
	viper.SetDefault("JWT_REFRESH_SECRET", "super_refresh_secret_123")
	viper.SetDefault("JWT_ACCESS_TTL_MIN", 15)
	viper.SetDefault("JWT_REFRESH_TTL_DAYS", 7)

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
			BodyLimitMB:   viper.GetInt("BODY_LIMIT_MB"),
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
		CORSAllow: viper.GetString("CORS_ALLOW_ORIGINS"),
		Shutdown:  time.Duration(viper.GetInt("SHUTDOWN_TIMEOUT_SEC")) * time.Second,
		JWT: JWTConfig{
			AccessSecret:  viper.GetString("JWT_ACCESS_SECRET"),
			RefreshSecret: viper.GetString("JWT_REFRESH_SECRET"),
		},
	}

	log.Info("Gateway - Configuration loaded")
	return nil
}
