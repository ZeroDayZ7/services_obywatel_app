package types

import "time"

type ServicesConfig struct {
	Auth      string
	Documents string
	Notify    string
	Users     string
}

type InternalSecurityConfig struct {
	HMACSecret string
}

type OTELConfig struct {
	Enabled     bool
	Endpoint    string
	ServiceName string
}

type DBConfig struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type ProxyConfig struct {
	MaxIdleConns        int
	IdleConnTimeout     time.Duration
	MaxIdleConnsPerHost int
	RequestTimeout      time.Duration
}

type SessionConfig struct {
	Prefix string
	TTL    time.Duration
}

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
	Session   SessionConfig
	Proxy     ProxyConfig
	CORSAllow string
	Shutdown  time.Duration
	JWT       JWTConfig
	OTEL      OTELConfig
	Internal  InternalSecurityConfig
	Services  ServicesConfig
	Database  DBConfig
}
