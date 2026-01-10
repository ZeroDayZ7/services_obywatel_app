package config

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/spf13/viper"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/pkg/types"
	pkgConfig "github.com/zerodayz7/platform/pkg/viper"
)

var (
	AppConfig types.Config
	Store     *session.Store
)

func LoadConfigGlobal() error {
	log := shared.GetLogger()

	pkgConfig.SetSharedDefaults("auth")

	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.ErrorObj("Error loading .env", err)
			return fmt.Errorf("error loading .env: %v", err)
		}
	}

	AppConfig = types.Config{
		Server: types.ServerConfig{
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
		Redis: types.RedisConfig{
			Host:     viper.GetString("REDIS_HOST"),
			Port:     viper.GetString("REDIS_PORT"),
			Password: viper.GetString("REDIS_PASSWORD"),
			DB:       viper.GetInt("REDIS_DB"),
		},
		Internal: types.InternalSecurityConfig{
			HMACSecret: viper.GetString("INTERNAL_HMAC_SECRET"),
		},
		Session: types.SessionConfig{
			Prefix: viper.GetString("REDIS_SESSION_PREFIX"),
			TTL:    time.Duration(viper.GetInt("REDIS_SESSION_TTL_MIN")) * time.Minute,
		},
		Database: types.DBConfig{
			DSN:             viper.GetString("DATABASE_DSN"),
			MaxOpenConns:    viper.GetInt("DB_MAX_OPEN_CONNS"),
			MaxIdleConns:    viper.GetInt("DB_MAX_IDLE_CONNS"),
			ConnMaxLifetime: time.Duration(viper.GetInt("DB_CONN_MAX_LIFETIME_MIN")) * time.Minute,
		},
		Shutdown: time.Duration(viper.GetInt("SHUTDOWN_TIMEOUT_SEC")) * time.Second,
		JWT: types.JWTConfig{
			AccessSecret:  viper.GetString("JWT_ACCESS_SECRET"),
			RefreshSecret: viper.GetString("JWT_REFRESH_SECRET"),
			AccessTTL:     time.Duration(viper.GetInt("JWT_ACCESS_TTL_MIN")) * time.Minute,
			RefreshTTL:    time.Duration(viper.GetInt("JWT_REFRESH_TTL_DAYS")) * 24 * time.Hour,
		},
		OTEL: types.OTELConfig{
			Enabled:     viper.GetBool("OTEL_ENABLED"),
			Endpoint:    viper.GetString("OTEL_ENDPOINT"),
			ServiceName: viper.GetString("OTEL_SERVICE_NAME"),
		},
	}

	if AppConfig.Internal.HMACSecret == "" {
		return fmt.Errorf("INTERNAL_HMAC_SECRET is required")
	}

	log.Info("Configuration loaded")
	return nil
}
