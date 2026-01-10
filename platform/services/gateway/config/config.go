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

	pkgConfig.SetSharedDefaults("gateway")

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
		Internal: types.InternalSecurityConfig{
			HMACSecret: viper.GetString("INTERNAL_HMAC_SECRET"),
		},
		Redis: types.RedisConfig{
			Host:     viper.GetString("REDIS_HOST"),
			Port:     viper.GetString("REDIS_PORT"),
			Password: viper.GetString("REDIS_PASSWORD"),
			DB:       viper.GetInt("REDIS_DB"),
		},
		Session: types.SessionConfig{
			Prefix: viper.GetString("REDIS_SESSION_PREFIX"),
			TTL:    time.Duration(viper.GetInt("REDIS_SESSION_TTL_MIN")) * time.Minute,
		},
		CORSAllow: viper.GetString("CORS_ALLOW_ORIGINS"),
		Shutdown:  time.Duration(viper.GetInt("SHUTDOWN_TIMEOUT_SEC")) * time.Second,
		JWT: types.JWTConfig{
			AccessSecret:  viper.GetString("JWT_ACCESS_SECRET"),
			RefreshSecret: viper.GetString("JWT_REFRESH_SECRET"),
		},

		Proxy: types.ProxyConfig{
			MaxIdleConns:        viper.GetInt("PROXY_MAX_IDLE_CONNS"),
			IdleConnTimeout:     time.Duration(viper.GetInt("PROXY_IDLE_CONN_TIMEOUT_SEC")) * time.Second,
			MaxIdleConnsPerHost: viper.GetInt("PROXY_MAX_IDLE_CONNS_PER_HOST"),
			RequestTimeout:      time.Duration(viper.GetInt("PROXY_REQUEST_TIMEOUT_SEC")) * time.Second,
		},
		OTEL: types.OTELConfig{
			Enabled:     viper.GetBool("OTEL_ENABLED"),
			Endpoint:    viper.GetString("OTEL_ENDPOINT"),
			ServiceName: viper.GetString("OTEL_SERVICE_NAME"),
		},

		Services: types.ServicesConfig{
			Auth:      viper.GetString("SERVICE_AUTH_URL"),
			Documents: viper.GetString("SERVICE_DOCS_URL"),
			Notify:    viper.GetString("SERVICE_NOTIFY_URL"),
			Users:     viper.GetString("SERVICE_USERS_URL"),
		},
	}

	if AppConfig.Internal.HMACSecret == "" {
		return fmt.Errorf("INTERNAL_HMAC_SECRET is required")
	}

	log.Info("Configuration loaded")
	return nil
}
