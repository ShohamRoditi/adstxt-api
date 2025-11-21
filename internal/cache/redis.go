package cache

import (
	"context"
	"time"

	"adstxt-api/internal/config"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client     *redis.Client
	defaultTTL time.Duration
	ctx        context.Context
}

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

func (rc *RedisCache) Get(key string) ([]byte, error) {
	val, err := rc.client.Get(rc.ctx, key).Bytes()
	if err == redis.Nil {
		return nil, ErrCacheNotFound
	}
	return val, err
}

func (rc *RedisCache) Set(key string, value []byte, ttl time.Duration) error {
	if ttl == 0 {
		ttl = rc.defaultTTL
	}
	return rc.client.Set(rc.ctx, key, value, ttl).Err()
}

func (rc *RedisCache) Delete(key string) error {
	return rc.client.Del(rc.ctx, key).Err()
}

func (rc *RedisCache) Close() error {
	return rc.client.Close()
}
