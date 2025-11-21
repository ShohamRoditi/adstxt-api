package cache

import (
	"errors"
	"time"

	"adstxt-api/internal/config"
)

var ErrCacheNotFound = errors.New("cache entry not found")

type Cache interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte, ttl time.Duration) error
	Delete(key string) error
	Close() error
}

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
