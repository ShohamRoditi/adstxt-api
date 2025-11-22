// Package config provides configuration management for the ads.txt API service.
// Configuration is loaded from environment variables with sensible defaults.
package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration values for the application.
// All values are loaded from environment variables with fallback defaults.
type Config struct {
	Port               string        // HTTP server port (default: 8080)
	CacheType          string        // Cache backend: memory, redis, or file (default: memory)
	CacheTTL           time.Duration // Cache entry time-to-live (default: 1h)
	RateLimitPerSecond int           // Rate limit per client per second (default: 10)
	RedisAddr          string        // Redis server address (default: localhost:6379)
	RedisPassword      string        // Redis password (default: empty)
	RedisDB            int           // Redis database number (default: 0)
	FileStoragePath    string        // File cache storage path (default: ./cache)
	RequestTimeout     time.Duration // HTTP request timeout (default: 10s)
}

// Load creates a new Config by reading environment variables.
// If an environment variable is not set or invalid, the default value is used.
func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "8080"),
		CacheType:          getEnv("CACHE_TYPE", "redis"),
		CacheTTL:           getDurationEnv("CACHE_TTL", 1*time.Hour),
		RateLimitPerSecond: getIntEnv("RATE_LIMIT_PER_SECOND", 10),
		RedisAddr:          getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:      getEnv("REDIS_PASSWORD", ""),
		RedisDB:            getIntEnv("REDIS_DB", 0),
		FileStoragePath:    getEnv("FILE_STORAGE_PATH", "./cache"),
		RequestTimeout:     getDurationEnv("REQUEST_TIMEOUT", 10*time.Second),
	}
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
