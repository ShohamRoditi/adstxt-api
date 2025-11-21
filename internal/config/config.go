package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port               string
	CacheType          string
	CacheTTL           time.Duration
	RateLimitPerSecond int
	RedisAddr          string
	RedisPassword      string
	RedisDB            int
	FileStoragePath    string
	RequestTimeout     time.Duration
}

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
