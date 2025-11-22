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
