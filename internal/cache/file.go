package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// fileCacheEntry represents a cache entry stored on disk as JSON.
// It contains the cached value and its expiration time.
type fileCacheEntry struct {
	Value      []byte    `json:"value"`
	Expiration time.Time `json:"expiration"`
}

// FileCache is a file-based cache implementation that stores data as JSON files on disk.
// Each cache entry is stored in a separate file within the specified base directory.
// It provides persistent caching across application restarts but is slower than memory-based caching.
// All methods are safe for concurrent use.
type FileCache struct {
	basePath   string
	defaultTTL time.Duration
	mu         sync.RWMutex
}

// NewFileCache creates a new FileCache with the specified base path and default TTL.
// It creates the base directory if it doesn't exist. Files are created with 0755 permissions.
// Returns an error if the directory cannot be created.
func NewFileCache(basePath string, defaultTTL time.Duration) (*FileCache, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, err
	}

	return &FileCache{
		basePath:   basePath,
		defaultTTL: defaultTTL,
	}, nil
}

// sanitizeKey creates a safe filename from a cache key by hashing it.
// This prevents path traversal attacks from malicious keys like "../../../etc/passwd"
func (fc *FileCache) sanitizeKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// Get retrieves a value from the file cache by reading the corresponding JSON file.
// Returns ErrCacheNotFound if the file doesn't exist or the entry has expired.
// The key is sanitized (hashed) to prevent path traversal attacks.
func (fc *FileCache) Get(key string) ([]byte, error) {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	safeKey := fc.sanitizeKey(key)
	filePath := filepath.Join(fc.basePath, safeKey+".json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, ErrCacheNotFound
	}

	var entry fileCacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}

	if time.Now().After(entry.Expiration) {
		return nil, ErrCacheNotFound
	}

	return entry.Value, nil
}

// Set stores a value in the file cache by writing it to a JSON file.
// If ttl is 0, the default TTL is used. The file is created with 0644 permissions.
// The key is sanitized (hashed) to prevent path traversal attacks.
func (fc *FileCache) Set(key string, value []byte, ttl time.Duration) error {
	if ttl == 0 {
		ttl = fc.defaultTTL
	}

	fc.mu.Lock()
	defer fc.mu.Unlock()

	entry := fileCacheEntry{
		Value:      value,
		Expiration: time.Now().Add(ttl),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	safeKey := fc.sanitizeKey(key)
	filePath := filepath.Join(fc.basePath, safeKey+".json")
	return os.WriteFile(filePath, data, 0644)
}

// Delete removes a cache entry by deleting its corresponding file.
// Returns an error if the file cannot be deleted (except when it doesn't exist).
// The key is sanitized (hashed) to prevent path traversal attacks.
func (fc *FileCache) Delete(key string) error {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	safeKey := fc.sanitizeKey(key)
	filePath := filepath.Join(fc.basePath, safeKey+".json")
	return os.Remove(filePath)
}

// Close is a no-op for FileCache as there are no persistent connections or resources to clean up.
// Implements the Cache interface.
func (fc *FileCache) Close() error {
	return nil
}
