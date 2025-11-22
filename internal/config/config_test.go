package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected Config
	}{
		{
			name:    "default values",
			envVars: map[string]string{},
			expected: Config{
				Port:               "8080",
				CacheType:          "memory",
				CacheTTL:           1 * time.Hour,
				RateLimitPerSecond: 10,
				RedisAddr:          "localhost:6379",
				RedisPassword:      "",
				RedisDB:            0,
				FileStoragePath:    "./cache",
				RequestTimeout:     10 * time.Second,
			},
		},
		{
			name: "custom values",
			envVars: map[string]string{
				"PORT":                  "9000",
				"CACHE_TYPE":            "redis",
				"CACHE_TTL":             "2h",
				"RATE_LIMIT_PER_SECOND": "20",
				"REDIS_ADDR":            "redis:6379",
				"REDIS_PASSWORD":        "secret",
				"REDIS_DB":              "1",
				"FILE_STORAGE_PATH":     "/tmp/cache",
				"REQUEST_TIMEOUT":       "30s",
			},
			expected: Config{
				Port:               "9000",
				CacheType:          "redis",
				CacheTTL:           2 * time.Hour,
				RateLimitPerSecond: 20,
				RedisAddr:          "redis:6379",
				RedisPassword:      "secret",
				RedisDB:            1,
				FileStoragePath:    "/tmp/cache",
				RequestTimeout:     30 * time.Second,
			},
		},
		{
			name: "invalid duration falls back to default",
			envVars: map[string]string{
				"CACHE_TTL":       "invalid",
				"REQUEST_TIMEOUT": "bad",
			},
			expected: Config{
				Port:               "8080",
				CacheType:          "memory",
				CacheTTL:           1 * time.Hour,
				RateLimitPerSecond: 10,
				RedisAddr:          "localhost:6379",
				RedisPassword:      "",
				RedisDB:            0,
				FileStoragePath:    "./cache",
				RequestTimeout:     10 * time.Second,
			},
		},
		{
			name: "invalid int falls back to default",
			envVars: map[string]string{
				"RATE_LIMIT_PER_SECOND": "invalid",
				"REDIS_DB":              "bad",
			},
			expected: Config{
				Port:               "8080",
				CacheType:          "memory",
				CacheTTL:           1 * time.Hour,
				RateLimitPerSecond: 10,
				RedisAddr:          "localhost:6379",
				RedisPassword:      "",
				RedisDB:            0,
				FileStoragePath:    "./cache",
				RequestTimeout:     10 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			os.Clearenv()

			// Set test environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			cfg := Load()

			if cfg.Port != tt.expected.Port {
				t.Errorf("Port = %v, want %v", cfg.Port, tt.expected.Port)
			}
			if cfg.CacheType != tt.expected.CacheType {
				t.Errorf("CacheType = %v, want %v", cfg.CacheType, tt.expected.CacheType)
			}
			if cfg.CacheTTL != tt.expected.CacheTTL {
				t.Errorf("CacheTTL = %v, want %v", cfg.CacheTTL, tt.expected.CacheTTL)
			}
			if cfg.RateLimitPerSecond != tt.expected.RateLimitPerSecond {
				t.Errorf("RateLimitPerSecond = %v, want %v", cfg.RateLimitPerSecond, tt.expected.RateLimitPerSecond)
			}
			if cfg.RedisAddr != tt.expected.RedisAddr {
				t.Errorf("RedisAddr = %v, want %v", cfg.RedisAddr, tt.expected.RedisAddr)
			}
			if cfg.RedisPassword != tt.expected.RedisPassword {
				t.Errorf("RedisPassword = %v, want %v", cfg.RedisPassword, tt.expected.RedisPassword)
			}
			if cfg.RedisDB != tt.expected.RedisDB {
				t.Errorf("RedisDB = %v, want %v", cfg.RedisDB, tt.expected.RedisDB)
			}
			if cfg.FileStoragePath != tt.expected.FileStoragePath {
				t.Errorf("FileStoragePath = %v, want %v", cfg.FileStoragePath, tt.expected.FileStoragePath)
			}
			if cfg.RequestTimeout != tt.expected.RequestTimeout {
				t.Errorf("RequestTimeout = %v, want %v", cfg.RequestTimeout, tt.expected.RequestTimeout)
			}
		})
	}
}

func TestGetEnv(t *testing.T) {
	os.Clearenv()

	// Test default
	result := getEnv("NON_EXISTENT", "default")
	if result != "default" {
		t.Errorf("getEnv() = %v, want %v", result, "default")
	}

	// Test existing
	os.Setenv("TEST_VAR", "value")
	result = getEnv("TEST_VAR", "default")
	if result != "value" {
		t.Errorf("getEnv() = %v, want %v", result, "value")
	}
}

func TestGetIntEnv(t *testing.T) {
	os.Clearenv()

	// Test default
	result := getIntEnv("NON_EXISTENT", 42)
	if result != 42 {
		t.Errorf("getIntEnv() = %v, want %v", result, 42)
	}

	// Test valid int
	os.Setenv("TEST_INT", "100")
	result = getIntEnv("TEST_INT", 42)
	if result != 100 {
		t.Errorf("getIntEnv() = %v, want %v", result, 100)
	}

	// Test invalid int
	os.Setenv("TEST_INT", "invalid")
	result = getIntEnv("TEST_INT", 42)
	if result != 42 {
		t.Errorf("getIntEnv() with invalid value = %v, want %v", result, 42)
	}
}

func TestGetDurationEnv(t *testing.T) {
	os.Clearenv()

	// Test default
	result := getDurationEnv("NON_EXISTENT", 5*time.Second)
	if result != 5*time.Second {
		t.Errorf("getDurationEnv() = %v, want %v", result, 5*time.Second)
	}

	// Test valid duration
	os.Setenv("TEST_DURATION", "1h")
	result = getDurationEnv("TEST_DURATION", 5*time.Second)
	if result != 1*time.Hour {
		t.Errorf("getDurationEnv() = %v, want %v", result, 1*time.Hour)
	}

	// Test invalid duration
	os.Setenv("TEST_DURATION", "invalid")
	result = getDurationEnv("TEST_DURATION", 5*time.Second)
	if result != 5*time.Second {
		t.Errorf("getDurationEnv() with invalid value = %v, want %v", result, 5*time.Second)
	}
}
