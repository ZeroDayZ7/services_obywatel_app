package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppName string
	Port    string
	Env     string
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		AppName: getEnv("APP_NAME", "service-template"),
		Port:    getEnv("PORT", "8080"),
		Env:     getEnv("ENV", "development"),
	}
}

func getEnv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
