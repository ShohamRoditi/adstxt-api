package cache

import (
	"context"
	"testing"
	"time"

	"adstxt-api/internal/config"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// TestRedisCache_SetAndGet tests basic set and get operations
func TestRedisCache_SetAndGet(t *testing.T) {
	// Start a mock Redis server
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	// Create config with mock Redis address
	cfg := &config.Config{
		RedisAddr:     mr.Addr(),
		RedisPassword: "",
		RedisDB:       0,
		CacheTTL:      5 * time.Minute,
	}

	cache, err := NewRedisCache(cfg)
	if err != nil {
		t.Fatalf("NewRedisCache() error = %v", err)
	}
	defer cache.Close()

	// Test Set and Get
	key := "test-key"
	value := []byte("test-value")

	err = cache.Set(key, value, 1*time.Minute)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	result, err := cache.Get(key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if string(result) != string(value) {
		t.Errorf("Get() = %s, want %s", result, value)
	}
}

// TestRedisCache_GetNonExistent tests getting a non-existent key
func TestRedisCache_GetNonExistent(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	cfg := &config.Config{
		RedisAddr:     mr.Addr(),
		RedisPassword: "",
		RedisDB:       0,
		CacheTTL:      5 * time.Minute,
	}

	cache, err := NewRedisCache(cfg)
	if err != nil {
		t.Fatalf("NewRedisCache() error = %v", err)
	}
	defer cache.Close()

	_, err = cache.Get("non-existent-key")
	if err != ErrCacheNotFound {
		t.Errorf("Get() error = %v, want %v", err, ErrCacheNotFound)
	}
}

// TestRedisCache_Expiration tests that entries expire after TTL
func TestRedisCache_Expiration(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	cfg := &config.Config{
		RedisAddr:     mr.Addr(),
		RedisPassword: "",
		RedisDB:       0,
		CacheTTL:      5 * time.Minute,
	}

	cache, err := NewRedisCache(cfg)
	if err != nil {
		t.Fatalf("NewRedisCache() error = %v", err)
	}
	defer cache.Close()

	key := "expiring-key"
	value := []byte("expiring-value")

	// Set with 1 second TTL
	err = cache.Set(key, value, 1*time.Second)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Immediately get should work
	_, err = cache.Get(key)
	if err != nil {
		t.Errorf("Get() immediate error = %v", err)
	}

	// Fast forward time in miniredis
	mr.FastForward(2 * time.Second)

	// Get after expiration should fail
	_, err = cache.Get(key)
	if err != ErrCacheNotFound {
		t.Errorf("Get() after expiration error = %v, want %v", err, ErrCacheNotFound)
	}
}

// TestRedisCache_Delete tests deleting a key
func TestRedisCache_Delete(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	cfg := &config.Config{
		RedisAddr:     mr.Addr(),
		RedisPassword: "",
		RedisDB:       0,
		CacheTTL:      5 * time.Minute,
	}

	cache, err := NewRedisCache(cfg)
	if err != nil {
		t.Fatalf("NewRedisCache() error = %v", err)
	}
	defer cache.Close()

	key := "delete-key"
	value := []byte("delete-value")

	// Set a value
	err = cache.Set(key, value, 1*time.Minute)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Delete it
	err = cache.Delete(key)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Get should fail after delete
	_, err = cache.Get(key)
	if err != ErrCacheNotFound {
		t.Errorf("Get() after delete error = %v, want %v", err, ErrCacheNotFound)
	}
}

// TestRedisCache_DeleteNonExistent tests deleting a non-existent key
func TestRedisCache_DeleteNonExistent(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	cfg := &config.Config{
		RedisAddr:     mr.Addr(),
		RedisPassword: "",
		RedisDB:       0,
		CacheTTL:      5 * time.Minute,
	}

	cache, err := NewRedisCache(cfg)
	if err != nil {
		t.Fatalf("NewRedisCache() error = %v", err)
	}
	defer cache.Close()

	// Delete non-existent key should not error
	err = cache.Delete("non-existent-key")
	if err != nil {
		t.Errorf("Delete() non-existent key error = %v, want nil", err)
	}
}

// TestRedisCache_DefaultTTL tests using default TTL when ttl=0
func TestRedisCache_DefaultTTL(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	defaultTTL := 10 * time.Minute
	cfg := &config.Config{
		RedisAddr:     mr.Addr(),
		RedisPassword: "",
		RedisDB:       0,
		CacheTTL:      defaultTTL,
	}

	cache, err := NewRedisCache(cfg)
	if err != nil {
		t.Fatalf("NewRedisCache() error = %v", err)
	}
	defer cache.Close()

	key := "default-ttl-key"
	value := []byte("default-ttl-value")

	// Set with ttl=0 to use default
	err = cache.Set(key, value, 0)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Check the TTL in Redis
	ttl := mr.TTL(key)
	if ttl != defaultTTL {
		t.Errorf("TTL = %v, want %v", ttl, defaultTTL)
	}
}

// TestRedisCache_ConnectionFailure tests handling of connection failures
func TestRedisCache_ConnectionFailure(t *testing.T) {
	cfg := &config.Config{
		RedisAddr:     "localhost:9999", // Non-existent Redis server
		RedisPassword: "",
		RedisDB:       0,
		CacheTTL:      5 * time.Minute,
	}

	_, err := NewRedisCache(cfg)
	if err == nil {
		t.Error("NewRedisCache() expected error for invalid address, got nil")
	}
}

// TestRedisCache_ConcurrentAccess tests concurrent get/set operations
func TestRedisCache_ConcurrentAccess(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	cfg := &config.Config{
		RedisAddr:     mr.Addr(),
		RedisPassword: "",
		RedisDB:       0,
		CacheTTL:      5 * time.Minute,
	}

	cache, err := NewRedisCache(cfg)
	if err != nil {
		t.Fatalf("NewRedisCache() error = %v", err)
	}
	defer cache.Close()

	// Concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			key := "concurrent-key"
			value := []byte("value")
			cache.Set(key, value, 1*time.Minute)
			cache.Get(key)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestRedisCache_Close tests that Close properly closes the connection
func TestRedisCache_Close(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	cfg := &config.Config{
		RedisAddr:     mr.Addr(),
		RedisPassword: "",
		RedisDB:       0,
		CacheTTL:      5 * time.Minute,
	}

	cache, err := NewRedisCache(cfg)
	if err != nil {
		t.Fatalf("NewRedisCache() error = %v", err)
	}

	// Close should not error
	err = cache.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Operations after close should fail
	err = cache.Set("key", []byte("value"), 1*time.Minute)
	if err == nil {
		t.Error("Set() after Close() should error, got nil")
	}
}

// TestRedisCache_LargeValue tests storing and retrieving large values
func TestRedisCache_LargeValue(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	cfg := &config.Config{
		RedisAddr:     mr.Addr(),
		RedisPassword: "",
		RedisDB:       0,
		CacheTTL:      5 * time.Minute,
	}

	cache, err := NewRedisCache(cfg)
	if err != nil {
		t.Fatalf("NewRedisCache() error = %v", err)
	}
	defer cache.Close()

	// Create a large value (1MB)
	largeValue := make([]byte, 1024*1024)
	for i := range largeValue {
		largeValue[i] = byte(i % 256)
	}

	key := "large-key"
	err = cache.Set(key, largeValue, 1*time.Minute)
	if err != nil {
		t.Fatalf("Set() large value error = %v", err)
	}

	result, err := cache.Get(key)
	if err != nil {
		t.Fatalf("Get() large value error = %v", err)
	}

	if len(result) != len(largeValue) {
		t.Errorf("Get() large value length = %d, want %d", len(result), len(largeValue))
	}
}

// TestNewRedisCache_WithPassword tests Redis connection with authentication
func TestNewRedisCache_WithPassword(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	// Set password on mock Redis
	mr.RequireAuth("secret-password")

	cfg := &config.Config{
		RedisAddr:     mr.Addr(),
		RedisPassword: "secret-password",
		RedisDB:       0,
		CacheTTL:      5 * time.Minute,
	}

	cache, err := NewRedisCache(cfg)
	if err != nil {
		t.Fatalf("NewRedisCache() with password error = %v", err)
	}
	defer cache.Close()

	// Test that operations work with correct password
	err = cache.Set("key", []byte("value"), 1*time.Minute)
	if err != nil {
		t.Errorf("Set() with password error = %v", err)
	}
}

// TestNewRedisCache_WrongPassword tests Redis connection with wrong password
func TestNewRedisCache_WrongPassword(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	// Set password on mock Redis
	mr.RequireAuth("secret-password")

	cfg := &config.Config{
		RedisAddr:     mr.Addr(),
		RedisPassword: "wrong-password",
		RedisDB:       0,
		CacheTTL:      5 * time.Minute,
	}

	// Should fail with wrong password
	_, err = NewRedisCache(cfg)
	if err == nil {
		t.Error("NewRedisCache() with wrong password should error, got nil")
	}
}

// TestRedisCache_MultipleDBs tests using different Redis databases
func TestRedisCache_MultipleDBs(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	// Create cache for DB 0
	cfg0 := &config.Config{
		RedisAddr:     mr.Addr(),
		RedisPassword: "",
		RedisDB:       0,
		CacheTTL:      5 * time.Minute,
	}

	cache0, err := NewRedisCache(cfg0)
	if err != nil {
		t.Fatalf("NewRedisCache() DB0 error = %v", err)
	}
	defer cache0.Close()

	// Create cache for DB 1
	cfg1 := &config.Config{
		RedisAddr:     mr.Addr(),
		RedisPassword: "",
		RedisDB:       1,
		CacheTTL:      5 * time.Minute,
	}

	cache1, err := NewRedisCache(cfg1)
	if err != nil {
		t.Fatalf("NewRedisCache() DB1 error = %v", err)
	}
	defer cache1.Close()

	// Set same key in both DBs
	key := "test-key"
	value0 := []byte("value-db0")
	value1 := []byte("value-db1")

	cache0.Set(key, value0, 1*time.Minute)
	cache1.Set(key, value1, 1*time.Minute)

	// Verify they are isolated
	result0, _ := cache0.Get(key)
	result1, _ := cache1.Get(key)

	if string(result0) != string(value0) {
		t.Errorf("DB0 Get() = %s, want %s", result0, value0)
	}

	if string(result1) != string(value1) {
		t.Errorf("DB1 Get() = %s, want %s", result1, value1)
	}
}

// TestRedisCache_ContextCancellation tests behavior with context cancellation
func TestRedisCache_ContextCancellation(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	cfg := &config.Config{
		RedisAddr:     mr.Addr(),
		RedisPassword: "",
		RedisDB:       0,
		CacheTTL:      5 * time.Minute,
	}

	// Create a RedisCache with a cancellable context
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	ctx, cancel := context.WithCancel(context.Background())

	cache := &RedisCache{
		client:     client,
		defaultTTL: cfg.CacheTTL,
		ctx:        ctx,
	}
	defer cache.Close()

	// Set a value (should work)
	err = cache.Set("key", []byte("value"), 1*time.Minute)
	if err != nil {
		t.Fatalf("Set() before cancel error = %v", err)
	}

	// Cancel the context
	cancel()

	// Operations should fail after context cancellation
	err = cache.Set("key2", []byte("value2"), 1*time.Minute)
	if err == nil {
		t.Error("Set() after context cancel should error, got nil")
	}
}
