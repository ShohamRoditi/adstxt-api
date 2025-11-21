package cache

import (
	"sync"
	"time"
)

type cacheEntry struct {
	value      []byte
	expiration time.Time
}

type MemoryCache struct {
	data       map[string]*cacheEntry
	mu         sync.RWMutex
	defaultTTL time.Duration
	cleanupT   *time.Ticker
}

func NewMemoryCache(defaultTTL time.Duration) *MemoryCache {
	mc := &MemoryCache{
		data:       make(map[string]*cacheEntry),
		defaultTTL: defaultTTL,
		cleanupT:   time.NewTicker(5 * time.Minute),
	}

	go mc.cleanup()
	return mc
}

func (mc *MemoryCache) Get(key string) ([]byte, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	entry, exists := mc.data[key]
	if !exists || time.Now().After(entry.expiration) {
		return nil, ErrCacheNotFound
	}

	return entry.value, nil
}

func (mc *MemoryCache) Set(key string, value []byte, ttl time.Duration) error {
	if ttl == 0 {
		ttl = mc.defaultTTL
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.data[key] = &cacheEntry{
		value:      value,
		expiration: time.Now().Add(ttl),
	}

	return nil
}

func (mc *MemoryCache) Delete(key string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	delete(mc.data, key)
	return nil
}

func (mc *MemoryCache) Close() error {
	mc.cleanupT.Stop()
	return nil
}

func (mc *MemoryCache) cleanup() {
	for range mc.cleanupT.C {
		mc.mu.Lock()
		now := time.Now()
		for key, entry := range mc.data {
			if now.After(entry.expiration) {
				delete(mc.data, key)
			}
		}
		mc.mu.Unlock()
	}
}
