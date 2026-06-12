package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration. Every field has a sensible
// default where possible; the few that don't (DATABASE_URL, JWT_SECRET)
// fail loudly at startup so we never silently run with broken settings.
type Config struct {
	DatabaseURL   string
	JWTSecret     string
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration
	Port          string
	AppleBundleID string

	// Env: "development", "staging", or "production". Drives gin's
	// release mode and the production-only CORS / secret checks.
	Env string

	// AllowedOrigins is the CORS allowlist. In production this MUST be
	// explicit — wildcard origins are rejected at startup.
	// Dev defaults to no allowlist, which the main router treats as
	// "allow any origin" so a Flutter web hot-reload doesn't break.
	AllowedOrigins []string

	// HTTP server timeouts. Without these, slow clients can pin
	// connections indefinitely (slowloris).
	HTTPReadTimeout       time.Duration
	HTTPReadHeaderTimeout time.Duration
	HTTPWriteTimeout      time.Duration
	HTTPIdleTimeout       time.Duration

	// MaxBodyBytes caps each incoming request body. 1 MiB is more than
	// enough for our JSON-only API.
	MaxBodyBytes int64

	// Postgres pool sizing. Tune for the deployment target — defaults
	// are safe for a small single-instance deployment.
	DBMaxConns int32
	DBMinConns int32
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	env := getEnv("ENV", "development")

	cfg := &Config{
		DatabaseURL:           os.Getenv("DATABASE_URL"),
		JWTSecret:             os.Getenv("JWT_SECRET"),
		JWTAccessTTL:          parseDuration(os.Getenv("JWT_ACCESS_TTL"), 15*time.Minute),
		JWTRefreshTTL:         parseDuration(os.Getenv("JWT_REFRESH_TTL"), 720*time.Hour),
		Port:                  getEnv("PORT", "8080"),
		AppleBundleID:         os.Getenv("APPLE_BUNDLE_ID"),
		Env:                   env,
		AllowedOrigins:        parseList(os.Getenv("ALLOWED_ORIGINS")),
		HTTPReadTimeout:       parseDuration(os.Getenv("HTTP_READ_TIMEOUT"), 15*time.Second),
		HTTPReadHeaderTimeout: parseDuration(os.Getenv("HTTP_READ_HEADER_TIMEOUT"), 5*time.Second),
		HTTPWriteTimeout:      parseDuration(os.Getenv("HTTP_WRITE_TIMEOUT"), 20*time.Second),
		HTTPIdleTimeout:       parseDuration(os.Getenv("HTTP_IDLE_TIMEOUT"), 60*time.Second),
		MaxBodyBytes:          parseInt64(os.Getenv("MAX_BODY_BYTES"), 1<<20), // 1 MiB
		DBMaxConns:            int32(parseInt64(os.Getenv("DB_MAX_CONNS"), 25)),
		DBMinConns:            int32(parseInt64(os.Getenv("DB_MIN_CONNS"), 2)),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// IsProduction returns true when the app is running with ENV=production.
// Used to switch gin into release mode and tighten error responses.
func (c *Config) IsProduction() bool { return c.Env == "production" }

func (c *Config) validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	// 32 bytes (256 bits) is the floor for HS256 per RFC 7518. We accept
	// any string ≥ 32 chars on the assumption it's been generated from a
	// CSPRNG — `openssl rand -hex 32` is the suggested recipe.
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters (got %d)", len(c.JWTSecret))
	}

	// In production we refuse wildcard CORS and require an explicit list.
	// Dev intentionally allows empty (treated as "*" in the router).
	if c.IsProduction() {
		if len(c.AllowedOrigins) == 0 {
			return fmt.Errorf("ALLOWED_ORIGINS is required in production")
		}
		for _, o := range c.AllowedOrigins {
			if o == "*" {
				return fmt.Errorf("wildcard origin (*) is not allowed in production")
			}
		}
		if c.AppleBundleID == "" {
			return fmt.Errorf("APPLE_BUNDLE_ID is required in production (Sign in with Apple is mandatory on the App Store)")
		}
	}

	if c.DBMaxConns < 1 {
		return fmt.Errorf("DB_MAX_CONNS must be >= 1")
	}
	if c.DBMinConns < 0 || c.DBMinConns > c.DBMaxConns {
		return fmt.Errorf("DB_MIN_CONNS must be between 0 and DB_MAX_CONNS")
	}

	return nil
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

func parseInt64(s string, fallback int64) int64 {
	if s == "" {
		return fallback
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fallback
	}
	return v
}

// parseList splits "a,b,c" → ["a","b","c"], trimming whitespace.
func parseList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
