package cache

import (
	"log"
	"sync"
	"time"
)

// cacheEntry represents a single entry in the memory cache with its value and expiration time.
type cacheEntry struct {
	value      []byte
	expiration time.Time
}

// MemoryCache is an in-memory cache implementation that stores data in a map with expiration times.
// It automatically cleans up expired entries every 5 minutes via a background goroutine.
// All methods are safe for concurrent use.
type MemoryCache struct {
	data       map[string]*cacheEntry
	mu         sync.RWMutex
	defaultTTL time.Duration
	cleanupT   *time.Ticker
}

// NewMemoryCache creates a new MemoryCache with the specified default TTL.
// It starts a background goroutine that cleans up expired entries every 5 minutes.
func NewMemoryCache(defaultTTL time.Duration) *MemoryCache {
	mc := &MemoryCache{
		data:       make(map[string]*cacheEntry),
		defaultTTL: defaultTTL,
		cleanupT:   time.NewTicker(5 * time.Minute),
	}

	go mc.cleanup()
	return mc
}

// Get retrieves a value from the cache by key.
// Returns ErrCacheNotFound if the key doesn't exist or has expired.
func (mc *MemoryCache) Get(key string) ([]byte, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	entry, exists := mc.data[key]
	if !exists || time.Now().After(entry.expiration) {
		return nil, ErrCacheNotFound
	}

	return entry.value, nil
}

// Set stores a value in the cache with the specified TTL.
// If ttl is 0, the default TTL is used. The entry will be automatically
// removed after it expires during the next cleanup cycle.
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

// Delete removes a key from the cache.
// Returns nil even if the key doesn't exist.
func (mc *MemoryCache) Delete(key string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	delete(mc.data, key)
	return nil
}

// Close stops the background cleanup goroutine and releases resources.
// Should be called when the cache is no longer needed to prevent goroutine leaks.
func (mc *MemoryCache) Close() error {
	mc.cleanupT.Stop()
	return nil
}

// cleanup is a background goroutine that removes expired entries every 5 minutes.
// It iterates through all entries and deletes those that have passed their expiration time.
func (mc *MemoryCache) cleanup() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in MemoryCache cleanup goroutine: %v", r)
		}
	}()

	for range mc.cleanupT.C {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("PANIC during MemoryCache cleanup iteration: %v", r)
				}
			}()

			mc.mu.Lock()
			defer mc.mu.Unlock()

			now := time.Now()
			deleted := 0
			for key, entry := range mc.data {
				if now.After(entry.expiration) {
					delete(mc.data, key)
					deleted++
				}
			}

			if deleted > 0 {
				log.Printf("MemoryCache cleanup: removed %d expired entries", deleted)
			}
		}()
	}
}
