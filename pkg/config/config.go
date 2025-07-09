// Package config provides configuration management for the Reservia API.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for our application.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Auth     AuthConfig
	Logger   LoggerConfig
}

// ServerConfig holds server configuration.
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DatabaseConfig holds database configuration.
type DatabaseConfig struct {
	URL            string
	MaxConnections int
	ConnectTimeout time.Duration
	QueryTimeout   time.Duration
}

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	JWTSecret     string
	TokenExpiry   time.Duration
	RefreshExpiry time.Duration
}

// LoggerConfig holds logging configuration.
type LoggerConfig struct {
	Level  string
	Format string
}

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Port:         getEnv("PORT", "8080"),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 10*time.Second),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:  getDurationEnv("SERVER_IDLE_TIMEOUT", 60*time.Second),
		},
		Database: DatabaseConfig{
			URL:            getEnv("DATABASE_URL", "mongodb://localhost:27017/reservia"),
			MaxConnections: getIntEnv("DB_MAX_CONNECTIONS", 100),
			ConnectTimeout: getDurationEnv("DB_CONNECT_TIMEOUT", 10*time.Second),
			QueryTimeout:   getDurationEnv("DB_QUERY_TIMEOUT", 30*time.Second),
		},
		Auth: AuthConfig{
			JWTSecret:     getEnv("JWT_SECRET", ""),
			TokenExpiry:   getDurationEnv("JWT_TOKEN_EXPIRY", 24*time.Hour),
			RefreshExpiry: getDurationEnv("JWT_REFRESH_EXPIRY", 7*24*time.Hour),
		},
		Logger: LoggerConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// validate validates the configuration.
func (c *Config) validate() error {
	if c.Auth.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}

	if c.Database.URL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	return nil
}

// getEnv gets an environment variable with a default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getIntEnv gets an integer environment variable with a default value.
func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// getDurationEnv gets a duration environment variable with a default value.
func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
