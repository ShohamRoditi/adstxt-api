package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileCache_SetAndGet(t *testing.T) {
	tmpDir := t.TempDir()

	fc, err := NewFileCache(tmpDir, 1*time.Hour)
	if err != nil {
		t.Fatalf("NewFileCache() error = %v", err)
	}
	defer fc.Close()

	key := "test-key"
	value := []byte("test-value")

	err = fc.Set(key, value, 0)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	got, err := fc.Get(key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if string(got) != string(value) {
		t.Errorf("Get() = %v, want %v", string(got), string(value))
	}
}

func TestFileCache_GetNonExistent(t *testing.T) {
	tmpDir := t.TempDir()

	fc, err := NewFileCache(tmpDir, 1*time.Hour)
	if err != nil {
		t.Fatalf("NewFileCache() error = %v", err)
	}
	defer fc.Close()

	_, err = fc.Get("non-existent")
	if err != ErrCacheNotFound {
		t.Errorf("Get() error = %v, want %v", err, ErrCacheNotFound)
	}
}

func TestFileCache_Expiration(t *testing.T) {
	tmpDir := t.TempDir()

	fc, err := NewFileCache(tmpDir, 1*time.Hour)
	if err != nil {
		t.Fatalf("NewFileCache() error = %v", err)
	}
	defer fc.Close()

	key := "expiring-key"
	value := []byte("expiring-value")

	// Set with 100ms TTL
	err = fc.Set(key, value, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Should exist immediately
	_, err = fc.Get(key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	// Wait for expiration
	time.Sleep(200 * time.Millisecond)

	// Should be expired
	_, err = fc.Get(key)
	if err != ErrCacheNotFound {
		t.Errorf("Get() after expiration error = %v, want %v", err, ErrCacheNotFound)
	}
}

func TestFileCache_Delete(t *testing.T) {
	tmpDir := t.TempDir()

	fc, err := NewFileCache(tmpDir, 1*time.Hour)
	if err != nil {
		t.Fatalf("NewFileCache() error = %v", err)
	}
	defer fc.Close()

	key := "delete-key"
	value := []byte("delete-value")

	err = fc.Set(key, value, 0)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	err = fc.Delete(key)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = fc.Get(key)
	if err != ErrCacheNotFound {
		t.Errorf("Get() after Delete() error = %v, want %v", err, ErrCacheNotFound)
	}
}

func TestFileCache_MultipleConcurrent(t *testing.T) {
	tmpDir := t.TempDir()

	fc, err := NewFileCache(tmpDir, 1*time.Hour)
	if err != nil {
		t.Fatalf("NewFileCache() error = %v", err)
	}
	defer fc.Close()

	done := make(chan bool)

	// Write goroutines
	for i := 0; i < 10; i++ {
		go func(n int) {
			key := filepath.Join("concurrent", string(rune(n)))
			value := []byte("value")
			_ = fc.Set(key, value, 0)
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < 10; i++ {
		<-done
	}

	// Read goroutines
	for i := 0; i < 10; i++ {
		go func(n int) {
			key := filepath.Join("concurrent", string(rune(n)))
			_, _ = fc.Get(key)
			done <- true
		}(i)
	}

	// Wait for all reads
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestFileCache_InvalidPath(t *testing.T) {
	// Use an invalid path (empty)
	_, err := NewFileCache("", 1*time.Hour)
	if err == nil {
		t.Error("NewFileCache() with empty path should return error")
	}
}

func TestFileCache_CreateDirectory(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "adstxt-test-cache")
	defer os.RemoveAll(tmpDir)

	fc, err := NewFileCache(tmpDir, 1*time.Hour)
	if err != nil {
		t.Fatalf("NewFileCache() error = %v", err)
	}
	defer fc.Close()

	// Check if directory was created
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Error("NewFileCache() did not create directory")
	}
}
