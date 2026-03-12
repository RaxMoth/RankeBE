package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL   string
	JWTSecret     string
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration
	Port          string
	AppleBundleID string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		JWTSecret:     os.Getenv("JWT_SECRET"),
		JWTAccessTTL:  parseDuration(os.Getenv("JWT_ACCESS_TTL"), 15*time.Minute),
		JWTRefreshTTL: parseDuration(os.Getenv("JWT_REFRESH_TTL"), 720*time.Hour),
		Port:          getEnv("PORT", "8080"),
		AppleBundleID: os.Getenv("APPLE_BUNDLE_ID"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseDuration(s string, fallback time.Duration) time.Duration {
	if s == "" {
		return fallback
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return fallback
	}
	return d
}
