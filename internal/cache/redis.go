package cache

import (
	"context"
	"time"

	"adstxt-api/internal/config"

	"github.com/redis/go-redis/v9"
)

// RedisCache is a Redis-based cache implementation that stores data in a Redis server.
// It provides distributed caching capabilities with automatic expiration.
// All methods are safe for concurrent use as they use the underlying Redis client's thread-safe operations.
type RedisCache struct {
	client     *redis.Client
	defaultTTL time.Duration
	ctx        context.Context
}

// NewRedisCache creates a new RedisCache using the configuration provided.
// It establishes a connection to the Redis server and verifies connectivity with a PING command.
// Returns an error if the Redis server is unreachable or authentication fails.
func NewRedisCache(cfg *config.Config) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &RedisCache{
		client:     client,
		defaultTTL: cfg.CacheTTL,
		ctx:        ctx,
	}, nil
}

// Get retrieves a value from Redis by key.
// Returns ErrCacheNotFound if the key doesn't exist or has expired.
// Redis handles expiration automatically, so expired keys are treated as not found.
func (rc *RedisCache) Get(key string) ([]byte, error) {
	val, err := rc.client.Get(rc.ctx, key).Bytes()
	if err == redis.Nil {
		return nil, ErrCacheNotFound
	}
	return val, err
}

// Set stores a value in Redis with the specified TTL.
// If ttl is 0, the default TTL is used. Redis will automatically remove the key after expiration.
func (rc *RedisCache) Set(key string, value []byte, ttl time.Duration) error {
	if ttl == 0 {
		ttl = rc.defaultTTL
	}
	return rc.client.Set(rc.ctx, key, value, ttl).Err()
}

// Delete removes a key from Redis.
// Returns nil even if the key doesn't exist.
func (rc *RedisCache) Delete(key string) error {
	return rc.client.Del(rc.ctx, key).Err()
}

// Close closes the Redis client connection and releases resources.
// Should be called when the cache is no longer needed to properly clean up connections.
func (rc *RedisCache) Close() error {
	return rc.client.Close()
}
