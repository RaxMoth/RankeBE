package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server
	Environment string
	Port        string
	
	// Database
	DatabaseType string // "mysql" or "firebase"
	
	// MySQL
	MySQLHost     string
	MySQLPort     string
	MySQLUser     string
	MySQLPassword string
	MySQLDatabase string
	
	// Firebase
	FirebaseProjectID     string
	FirebaseCredentials   string // Path to credentials file
	
	// JWT
	JWTSecret           string
	JWTExpiration       time.Duration
	JWTRefreshExpiration time.Duration
	
	// Rate Limiting
	RateLimitRequests  int
	RateLimitDuration  time.Duration
	
	// Logging
	LogLevel string
}

func Load() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	cfg := &Config{
		Environment:           getEnv("ENVIRONMENT", "development"),
		Port:                  getEnv("PORT", "8080"),
		DatabaseType:          getEnv("DATABASE_TYPE", "mysql"),
		
		// MySQL
		MySQLHost:             getEnv("MYSQL_HOST", "localhost"),
		MySQLPort:             getEnv("MYSQL_PORT", "3306"),
		MySQLUser:             getEnv("MYSQL_USER", "root"),
		MySQLPassword:         getEnv("MYSQL_PASSWORD", ""),
		MySQLDatabase:         getEnv("MYSQL_DATABASE", "gin_rest_db"),
		
		// Firebase
		FirebaseProjectID:     getEnv("FIREBASE_PROJECT_ID", ""),
		FirebaseCredentials:   getEnv("FIREBASE_CREDENTIALS", ""),
		
		// JWT
		JWTSecret:             getEnv("JWT_SECRET", "your-secret-key-change-this"),
		JWTExpiration:         getDurationEnv("JWT_EXPIRATION", 15*time.Minute),
		JWTRefreshExpiration:  getDurationEnv("JWT_REFRESH_EXPIRATION", 7*24*time.Hour),
		
		// Rate Limiting
		RateLimitRequests:     getIntEnv("RATE_LIMIT_REQUESTS", 100),
		RateLimitDuration:     getDurationEnv("RATE_LIMIT_DURATION", 1*time.Minute),
		
		// Logging
		LogLevel:              getEnv("LOG_LEVEL", "info"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.DatabaseType != "mysql" && c.DatabaseType != "firebase" {
		return fmt.Errorf("invalid database type: %s (must be 'mysql' or 'firebase')", c.DatabaseType)
	}

	if c.DatabaseType == "mysql" {
		if c.MySQLUser == "" || c.MySQLDatabase == "" {
			return fmt.Errorf("MySQL configuration incomplete")
		}
	}

	if c.DatabaseType == "firebase" {
		if c.FirebaseProjectID == "" || c.FirebaseCredentials == "" {
			return fmt.Errorf("Firebase configuration incomplete")
		}
	}

	if c.JWTSecret == "your-secret-key-change-this" {
		fmt.Println("WARNING: Using default JWT secret. Please change it in production!")
	}

	return nil
}

func (c *Config) GetMySQLDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.MySQLUser,
		c.MySQLPassword,
		c.MySQLHost,
		c.MySQLPort,
		c.MySQLDatabase,
	)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
