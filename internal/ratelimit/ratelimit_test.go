package ratelimit

import (
	"sync"
	"testing"
	"time"
)

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter(5)
	defer rl.Stop()

	clientID := "test-client"

	// Should allow first 5 requests
	for i := 0; i < 5; i++ {
		if !rl.Allow(clientID) {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 6th request should be blocked
	if rl.Allow(clientID) {
		t.Error("Request 6 should be blocked")
	}
}

func TestRateLimiter_Reset(t *testing.T) {
	rl := NewRateLimiter(2)
	defer rl.Stop()

	clientID := "test-client"

	// Use up tokens
	rl.Allow(clientID)
	rl.Allow(clientID)

	// Should be blocked
	if rl.Allow(clientID) {
		t.Error("Should be blocked before reset")
	}

	// Wait for window to reset
	time.Sleep(1100 * time.Millisecond)

	// Should allow again
	if !rl.Allow(clientID) {
		t.Error("Should allow after reset")
	}
}

func TestRateLimiter_MultipleClients(t *testing.T) {
	rl := NewRateLimiter(3)
	defer rl.Stop()

	client1 := "client-1"
	client2 := "client-2"

	// Both clients should have independent limits
	for i := 0; i < 3; i++ {
		if !rl.Allow(client1) {
			t.Errorf("Client1 request %d should be allowed", i+1)
		}
		if !rl.Allow(client2) {
			t.Errorf("Client2 request %d should be allowed", i+1)
		}
	}

	// Both should be blocked now
	if rl.Allow(client1) {
		t.Error("Client1 should be blocked")
	}
	if rl.Allow(client2) {
		t.Error("Client2 should be blocked")
	}
}

func TestRateLimiter_Concurrent(t *testing.T) {
	rl := NewRateLimiter(100)
	defer rl.Stop()

	clientID := "test-client"
	var wg sync.WaitGroup
	allowedCount := 0
	var mu sync.Mutex

	// Spawn 150 concurrent requests
	for i := 0; i < 150; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if rl.Allow(clientID) {
				mu.Lock()
				allowedCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Should allow exactly 100 requests
	if allowedCount != 100 {
		t.Errorf("Expected 100 allowed requests, got %d", allowedCount)
	}
}
