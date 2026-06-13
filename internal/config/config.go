package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPAddr                string
	ReadTimeout             time.Duration
	WriteTimeout            time.Duration
	ShutdownTimeout         time.Duration
	LogLevel                string
	DatabaseURL             string
	DatabaseConnectAttempts int
	DatabaseConnectDelay    time.Duration
	AutoMigrate             bool
	AuthAccessSecret        string
	AuthRefreshSecret       string
	AuthAccessTTL           time.Duration
	AuthRefreshTTL          time.Duration
	CORSOrigins             []string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		HTTPAddr:                getEnv("APP_HTTP_ADDR", ":8080"),
		ReadTimeout:             getDuration("APP_READ_TIMEOUT", 15*time.Second),
		WriteTimeout:            getDuration("APP_WRITE_TIMEOUT", 15*time.Second),
		ShutdownTimeout:         getDuration("APP_SHUTDOWN_TIMEOUT", 10*time.Second),
		LogLevel:                getEnv("APP_LOG_LEVEL", "info"),
		DatabaseURL:             os.Getenv("DATABASE_URL"),
		DatabaseConnectAttempts: getInt("DATABASE_CONNECT_ATTEMPTS", 15),
		DatabaseConnectDelay:    getDuration("DATABASE_CONNECT_DELAY", 2*time.Second),
		AutoMigrate:             getBool("APP_AUTO_MIGRATE", true),
		AuthAccessSecret:        os.Getenv("AUTH_ACCESS_SECRET"),
		AuthRefreshSecret:       os.Getenv("AUTH_REFRESH_SECRET"),
		AuthAccessTTL:           getDuration("AUTH_ACCESS_TTL", 15*time.Minute),
		AuthRefreshTTL:          getDuration("AUTH_REFRESH_TTL", 720*time.Hour),
		CORSOrigins:             getStringSlice("APP_CORS_ORIGINS", []string{"http://localhost:3000"}),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.AuthAccessSecret == "" {
		return nil, fmt.Errorf("AUTH_ACCESS_SECRET is required")
	}
	if cfg.AuthRefreshSecret == "" {
		return nil, fmt.Errorf("AUTH_REFRESH_SECRET is required")
	}

	return cfg, nil
}

func getEnv(name, fallback string) string {
	if value, ok := os.LookupEnv(name); ok {
		return value
	}
	return fallback
}

func getDuration(name string, fallback time.Duration) time.Duration {
	raw, ok := os.LookupEnv(name)
	if !ok || raw == "" {
		return fallback
	}

	duration, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}

	return duration
}

func getInt(name string, fallback int) int {
	raw, ok := os.LookupEnv(name)
	if !ok || raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}

	return value
}

func getBool(name string, fallback bool) bool {
	raw, ok := os.LookupEnv(name)
	if !ok || raw == "" {
		return fallback
	}

	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}

	return value
}

func getStringSlice(name string, fallback []string) []string {
	raw, ok := os.LookupEnv(name)
	if !ok || raw == "" {
		return fallback
	}

	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return fallback
	}
	return result
}
