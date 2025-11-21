package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type fileCacheEntry struct {
	Value      []byte    `json:"value"`
	Expiration time.Time `json:"expiration"`
}

type FileCache struct {
	basePath   string
	defaultTTL time.Duration
	mu         sync.RWMutex
}

func NewFileCache(basePath string, defaultTTL time.Duration) (*FileCache, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, err
	}

	return &FileCache{
		basePath:   basePath,
		defaultTTL: defaultTTL,
	}, nil
}

func (fc *FileCache) Get(key string) ([]byte, error) {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	filePath := filepath.Join(fc.basePath, key+".json")
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

	filePath := filepath.Join(fc.basePath, key+".json")
	return os.WriteFile(filePath, data, 0644)
}

func (fc *FileCache) Delete(key string) error {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	filePath := filepath.Join(fc.basePath, key+".json")
	return os.Remove(filePath)
}

func (fc *FileCache) Close() error {
	return nil
}
