package config

import (
	"os"
	"time"
)

type Config struct {
	Port         string
	TemplatesDir string
	AssetsDir    string
	ChromeBin    string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func Load() Config {
	return Config{
		Port:         getEnv("PORT", "8080"),
		TemplatesDir: getEnv("TEMPLATES_DIR", "./templates"),
		AssetsDir:    getEnv("ASSETS_DIR", "./assets"),
		ChromeBin:    getEnv("CHROME_BIN", "/usr/bin/chromium-browser"),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}
