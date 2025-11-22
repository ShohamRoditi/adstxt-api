// Package ratelimit provides a token bucket rate limiter implementation
// with per-client tracking and automatic cleanup of inactive clients.
package ratelimit

import (
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiting algorithm with per-client tracking.
// It is safe for concurrent use and automatically cleans up inactive clients.
type RateLimiter struct {
	limit    int
	window   time.Duration
	clients  map[string]*clientBucket
	mu       sync.RWMutex
	cleanupT *time.Ticker
}

// clientBucket represents a token bucket for a single client.
// Each client has their own bucket with a fixed number of tokens that refill over time.
type clientBucket struct {
	tokens    int
	lastReset time.Time
	mu        sync.Mutex
}

// NewRateLimiter creates a new RateLimiter with the specified limit per second.
// It starts a background goroutine that cleans up inactive clients every minute.
// Clients that have been inactive for more than 5 minutes are removed.
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

// Allow checks if a request from the given client should be allowed based on the rate limit.
// It returns true if the client has available tokens, false otherwise.
// The method is thread-safe and uses a token bucket algorithm where tokens are refilled
// every second based on the configured limit.
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
		now := time.Now()
		toDelete := []string{}

		rl.mu.RLock() // Use read lock first
		for clientID, bucket := range rl.clients {
			bucket.mu.Lock()
			if now.Sub(bucket.lastReset) > 5*time.Minute {
				toDelete = append(toDelete, clientID)
			}
			bucket.mu.Unlock()
		}
		rl.mu.RUnlock()

		// Now delete with write lock
		if len(toDelete) > 0 {
			rl.mu.Lock()
			for _, clientID := range toDelete {
				delete(rl.clients, clientID)
			}
			rl.mu.Unlock()
		}
	}
}

// Stop terminates the background cleanup goroutine.
// It should be called when the RateLimiter is no longer needed to prevent goroutine leaks.
func (rl *RateLimiter) Stop() {
	rl.cleanupT.Stop()
}
