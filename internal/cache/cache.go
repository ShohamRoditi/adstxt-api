// Package cache provides a pluggable caching interface with multiple backend implementations
// including in-memory, Redis, and file-based storage.
package cache

import (
	"errors"
	"time"

	"adstxt-api/internal/config"
)

// ErrCacheNotFound is returned when a cache key is not found or has expired.
var ErrCacheNotFound = errors.New("cache entry not found")

// Cache defines the interface for cache implementations.
// All methods are expected to be safe for concurrent use.
type Cache interface {
	// Get retrieves a value from the cache. Returns ErrCacheNotFound if the key doesn't exist or has expired.
	Get(key string) ([]byte, error)

	// Set stores a value in the cache with the specified TTL. A TTL of 0 uses the default TTL.
	Set(key string, value []byte, ttl time.Duration) error

	// Delete removes a key from the cache. Returns nil if the key doesn't exist.
	Delete(key string) error

	// Close releases any resources held by the cache implementation.
	Close() error
}

// NewCache creates a new Cache instance based on the specified type.
// Supported types: "memory", "redis", "file". Defaults to "memory" for unknown types.
func NewCache(cacheType string, cfg *config.Config) (Cache, error) {
	switch cacheType {
	case "memory":
		return NewMemoryCache(cfg.CacheTTL), nil
	case "redis":
		return NewRedisCache(cfg)
	case "file":
		return NewFileCache(cfg.FileStoragePath, cfg.CacheTTL)
	default:
		return NewMemoryCache(cfg.CacheTTL), nil
	}
}
