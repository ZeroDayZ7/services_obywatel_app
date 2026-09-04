package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port                string
	TemplatesDir        string
	AssetsDir           string
	ChromeBin           string
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
	PDFRenderTimeout    time.Duration
	PDFMaxConcurrency   int
	MaxRequestBodyBytes int64
}

func Load() Config {
	return Config{
		Port:                getEnv("PORT", "8080"),
		TemplatesDir:        getEnv("TEMPLATES_DIR", "./templates"),
		AssetsDir:           getEnv("ASSETS_DIR", "./assets"),
		ChromeBin:           getEnv("CHROME_BIN", "/usr/bin/chromium-browser"),
		ReadTimeout:         15 * time.Second,
		WriteTimeout:        60 * time.Second,
		PDFRenderTimeout:    getEnvDuration("PDF_RENDER_TIMEOUT", 30*time.Second),
		PDFMaxConcurrency:   getEnvInt("PDF_MAX_CONCURRENCY", 4),
		MaxRequestBodyBytes: getEnvInt64("MAX_REQUEST_BODY_BYTES", 2*1024*1024), // 2MB
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if valStr, ok := os.LookupEnv(key); ok && valStr != "" {
		if val, err := strconv.Atoi(valStr); err == nil {
			return val
		}
	}
	return fallback
}

func getEnvInt64(key string, fallback int64) int64 {
	if valStr, ok := os.LookupEnv(key); ok && valStr != "" {
		if val, err := strconv.ParseInt(valStr, 10, 64); err == nil {
			return val
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if valStr, ok := os.LookupEnv(key); ok && valStr != "" {
		if val, err := time.ParseDuration(valStr); err == nil {
			return val
		}
	}
	return fallback
}
