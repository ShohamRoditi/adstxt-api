package cache

import (
	"testing"
	"time"
)

func TestMemoryCache_SetGet(t *testing.T) {
	cache := NewMemoryCache(1 * time.Hour)
	defer cache.Close()

	key := "test-key"
	value := []byte("test-value")

	err := cache.Set(key, value, 0)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	retrieved, err := cache.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if string(retrieved) != string(value) {
		t.Errorf("Expected %s, got %s", value, retrieved)
	}
}

func TestMemoryCache_NotFound(t *testing.T) {
	cache := NewMemoryCache(1 * time.Hour)
	defer cache.Close()

	_, err := cache.Get("non-existent")
	if err != ErrCacheNotFound {
		t.Errorf("Expected ErrCacheNotFound, got %v", err)
	}
}

func TestMemoryCache_Expiration(t *testing.T) {
	cache := NewMemoryCache(1 * time.Hour)
	defer cache.Close()

	key := "test-key"
	value := []byte("test-value")

	// Set with 100ms TTL
	err := cache.Set(key, value, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Should be available immediately
	_, err = cache.Get(key)
	if err != nil {
		t.Error("Should be available before expiration")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	_, err = cache.Get(key)
	if err != ErrCacheNotFound {
		t.Error("Should be expired")
	}
}

func TestMemoryCache_Delete(t *testing.T) {
	cache := NewMemoryCache(1 * time.Hour)
	defer cache.Close()

	key := "test-key"
	value := []byte("test-value")

	_ = cache.Set(key, value, 0)

	err := cache.Delete(key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = cache.Get(key)
	if err != ErrCacheNotFound {
		t.Error("Key should be deleted")
	}
}

func TestMemoryCache_Cleanup(t *testing.T) {
	cache := NewMemoryCache(1 * time.Hour)
	defer cache.Close()

	// Add some entries with short TTL
	for i := 0; i < 5; i++ {
		key := string(rune('a' + i))
		cache.Set(key, []byte("value"), 50*time.Millisecond)
	}

	// Add one entry with long TTL
	cache.Set("long-lived", []byte("value"), 1*time.Hour)

	// Verify all entries exist
	cache.mu.RLock()
	initialCount := len(cache.data)
	cache.mu.RUnlock()

	if initialCount != 6 {
		t.Errorf("Expected 6 entries, got %d", initialCount)
	}

	// Wait for short TTL entries to expire
	time.Sleep(100 * time.Millisecond)

	// Manually trigger cleanup (since cleanup runs every 5 minutes)
	cache.mu.Lock()
	now := time.Now()
	deleted := 0
	for key, entry := range cache.data {
		if now.After(entry.expiration) {
			delete(cache.data, key)
			deleted++
		}
	}
	cache.mu.Unlock()

	if deleted != 5 {
		t.Errorf("Expected to delete 5 expired entries, deleted %d", deleted)
	}

	// Verify only long-lived entry remains
	cache.mu.RLock()
	finalCount := len(cache.data)
	cache.mu.RUnlock()

	if finalCount != 1 {
		t.Errorf("Expected 1 entry after cleanup, got %d", finalCount)
	}

	// Verify long-lived entry still accessible
	_, err := cache.Get("long-lived")
	if err != nil {
		t.Error("Long-lived entry should still be accessible")
	}
}

func TestMemoryCache_ConcurrentAccess(t *testing.T) {
	cache := NewMemoryCache(1 * time.Hour)
	defer cache.Close()

	// Test concurrent reads and writes don't cause race conditions
	done := make(chan bool)

	go func() {
		for i := 0; i < 100; i++ {
			cache.Set("key", []byte("value"), 1*time.Hour)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			cache.Get("key")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			cache.Delete("key")
		}
		done <- true
	}()

	// Wait for all goroutines
	<-done
	<-done
	<-done
}

func TestMemoryCache_Close(t *testing.T) {
	cache := NewMemoryCache(1 * time.Hour)

	cache.Set("key", []byte("value"), 0)

	// Close should not panic
	err := cache.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Should still be able to use cache after close (only cleanup stops)
	err = cache.Set("key2", []byte("value2"), 0)
	if err != nil {
		t.Error("Should still be able to use cache after Close()")
	}
}
