package ratelimit

import (
	"sync"
	"time"
)

type RateLimiter struct {
	limit    int
	window   time.Duration
	clients  map[string]*clientBucket
	mu       sync.RWMutex
	cleanupT *time.Ticker
}

type clientBucket struct {
	tokens    int
	lastReset time.Time
	mu        sync.Mutex
}

func NewRateLimiter(limitPerSecond int) *RateLimiter {
	rl := &RateLimiter{
		limit:   limitPerSecond,
		window:  time.Second,
		clients: make(map[string]*clientBucket),
	}

	rl.cleanupT = time.NewTicker(1 * time.Minute)
	go rl.cleanup()

	return rl
}

func (rl *RateLimiter) Allow(clientID string) bool {
	rl.mu.RLock()
	bucket, exists := rl.clients[clientID]
	rl.mu.RUnlock()

	if !exists {
		bucket = &clientBucket{
			tokens:    rl.limit,
			lastReset: time.Now(),
		}
		rl.mu.Lock()
		rl.clients[clientID] = bucket
		rl.mu.Unlock()
	}

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	now := time.Now()
	if now.Sub(bucket.lastReset) >= rl.window {
		bucket.tokens = rl.limit
		bucket.lastReset = now
	}

	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}

	return false
}

func (rl *RateLimiter) cleanup() {
	for range rl.cleanupT.C {
		rl.mu.Lock()
		now := time.Now()
		for clientID, bucket := range rl.clients {
			bucket.mu.Lock()
			if now.Sub(bucket.lastReset) > 5*time.Minute {
				delete(rl.clients, clientID)
			}
			bucket.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) Stop() {
	rl.cleanupT.Stop()
}
