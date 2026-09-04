package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Env                 string
	Port                string
	TemplatesDir        string
	AssetsDir           string
	ChromeBin           string
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
	PDFRenderTimeout    time.Duration
	ShutdownTimeout     time.Duration
	PDFMaxConcurrency   int
	MaxRequestBodyBytes int64
}

func Load() (Config, error) {
	cfg := Config{
		Env:                 getEnv("ENV", getEnv("APP_ENV", "development")),
		Port:                getEnv("PORT", "8080"),
		TemplatesDir:        getEnv("TEMPLATES_DIR", "./templates"),
		AssetsDir:           getEnv("ASSETS_DIR", "./assets"),
		ChromeBin:           getEnv("CHROME_BIN", "/usr/bin/chromium-browser"),
		ReadTimeout:         15 * time.Second,
		WriteTimeout:        60 * time.Second,
		PDFRenderTimeout:    30 * time.Second,
		ShutdownTimeout:     10 * time.Second,
		PDFMaxConcurrency:   4,
		MaxRequestBodyBytes: 2 * 1024 * 1024, // 2MB
	}

	var err error

	if val := os.Getenv("PDF_MAX_CONCURRENCY"); val != "" {
		cfg.PDFMaxConcurrency, err = strconv.Atoi(val)
		if err != nil || cfg.PDFMaxConcurrency <= 0 {
			return Config{}, fmt.Errorf("invalid PDF_MAX_CONCURRENCY value '%s': must be a positive integer", val)
		}
	}

	if val := os.Getenv("MAX_REQUEST_BODY_BYTES"); val != "" {
		cfg.MaxRequestBodyBytes, err = strconv.ParseInt(val, 10, 64)
		if err != nil || cfg.MaxRequestBodyBytes <= 0 {
			return Config{}, fmt.Errorf("invalid MAX_REQUEST_BODY_BYTES value '%s': must be a positive integer", val)
		}
	}

	if val := os.Getenv("PDF_RENDER_TIMEOUT"); val != "" {
		cfg.PDFRenderTimeout, err = time.ParseDuration(val)
		if err != nil || cfg.PDFRenderTimeout <= 0 {
			return Config{}, fmt.Errorf("invalid PDF_RENDER_TIMEOUT value '%s': must be a valid duration", val)
		}
	}

	if val := os.Getenv("SHUTDOWN_TIMEOUT"); val != "" {
		cfg.ShutdownTimeout, err = time.ParseDuration(val)
		if err != nil || cfg.ShutdownTimeout <= 0 {
			return Config{}, fmt.Errorf("invalid SHUTDOWN_TIMEOUT value '%s': must be a valid duration", val)
		}
	}

	if val := os.Getenv("READ_TIMEOUT"); val != "" {
		cfg.ReadTimeout, err = time.ParseDuration(val)
		if err != nil || cfg.ReadTimeout <= 0 {
			return Config{}, fmt.Errorf("invalid READ_TIMEOUT value '%s': must be a valid duration", val)
		}
	}

	if val := os.Getenv("WRITE_TIMEOUT"); val != "" {
		cfg.WriteTimeout, err = time.ParseDuration(val)
		if err != nil || cfg.WriteTimeout <= 0 {
			return Config{}, fmt.Errorf("invalid WRITE_TIMEOUT value '%s': must be a valid duration", val)
		}
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}
