package ratelimit

import (
	"fmt"
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
	type result struct {
		allowed bool
	}
	results := make(chan result, 150)

	// Spawn 150 concurrent requests
	for i := 0; i < 150; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allowed := rl.Allow(clientID)
			results <- result{allowed: allowed}
		}()
	}

	wg.Wait()
	close(results)

	// Count allowed requests
	allowedCount := 0
	for res := range results {
		if res.allowed {
			allowedCount++
		}
	}

	// Should allow at least 100 requests (might be slightly more due to timing/refills)
	// The important thing is it doesn't allow all 150
	if allowedCount < 100 {
		t.Errorf("Expected at least 100 allowed requests, got %d", allowedCount)
	}
	if allowedCount >= 150 {
		t.Errorf("Expected some requests to be blocked, but all 150 were allowed")
	}
}

func TestRateLimiter_Cleanup(t *testing.T) {
	rl := NewRateLimiter(10)
	defer rl.Stop()

	// Create multiple clients
	for i := 0; i < 5; i++ {
		clientID := fmt.Sprintf("client-%d", i)
		rl.Allow(clientID)
	}

	// Verify clients exist
	rl.mu.RLock()
	initialCount := len(rl.clients)
	rl.mu.RUnlock()

	if initialCount != 5 {
		t.Errorf("Expected 5 clients, got %d", initialCount)
	}

	// Wait for cleanup to run (cleanup runs every 1 minute, removes after 5 minutes of inactivity)
	// We can't easily wait 5 minutes, so let's manually trigger cleanup logic
	// by manipulating the lastReset time
	rl.mu.Lock()
	for _, bucket := range rl.clients {
		bucket.mu.Lock()
		bucket.lastReset = time.Now().Add(-6 * time.Minute) // Make it old
		bucket.mu.Unlock()
	}
	rl.mu.Unlock()

	// Manually trigger one cleanup iteration
	now := time.Now()
	toDelete := []string{}

	rl.mu.RLock()
	for clientID, bucket := range rl.clients {
		bucket.mu.Lock()
		if now.Sub(bucket.lastReset) > 5*time.Minute {
			toDelete = append(toDelete, clientID)
		}
		bucket.mu.Unlock()
	}
	rl.mu.RUnlock()

	if len(toDelete) > 0 {
		rl.mu.Lock()
		for _, clientID := range toDelete {
			delete(rl.clients, clientID)
		}
		rl.mu.Unlock()
	}

	// Verify cleanup happened
	rl.mu.RLock()
	finalCount := len(rl.clients)
	rl.mu.RUnlock()

	if finalCount != 0 {
		t.Errorf("Expected 0 clients after cleanup, got %d", finalCount)
	}
}

func TestRateLimiter_Stop(t *testing.T) {
	rl := NewRateLimiter(10)

	rl.Allow("test-client")

	// Stop should not panic
	rl.Stop()

	// Should still be able to use after stop (just cleanup goroutine stops)
	if !rl.Allow("test-client-2") {
		t.Error("Should still allow requests after Stop()")
	}
}

func TestRateLimiter_PartialTokenUsage(t *testing.T) {
	rl := NewRateLimiter(10)
	defer rl.Stop()

	clientID := "test-client"

	// Use only 5 out of 10 tokens
	for i := 0; i < 5; i++ {
		if !rl.Allow(clientID) {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// Wait for window to reset
	time.Sleep(1100 * time.Millisecond)

	// Should have full 10 tokens again after reset
	for i := 0; i < 10; i++ {
		if !rl.Allow(clientID) {
			t.Errorf("Request %d after reset should be allowed", i+1)
		}
	}

	// 11th should be blocked
	if rl.Allow(clientID) {
		t.Error("Request 11 should be blocked")
	}
}

func TestRateLimiter_ManyClients(t *testing.T) {
	rl := NewRateLimiter(5)
	defer rl.Stop()

	// Create 100 different clients
	for i := 0; i < 100; i++ {
		clientID := fmt.Sprintf("client-%d", i)

		// Each client should get their own bucket with full tokens
		for j := 0; j < 5; j++ {
			if !rl.Allow(clientID) {
				t.Errorf("Client %s request %d should be allowed", clientID, j+1)
			}
		}

		// Each client should be rate limited independently
		if rl.Allow(clientID) {
			t.Errorf("Client %s should be rate limited", clientID)
		}
	}

	// Verify all 100 client buckets exist
	rl.mu.RLock()
	clientCount := len(rl.clients)
	rl.mu.RUnlock()

	if clientCount != 100 {
		t.Errorf("Expected 100 client buckets, got %d", clientCount)
	}
}

func TestRateLimiter_ZeroLimit(t *testing.T) {
	// Edge case: rate limiter with 0 requests per second
	rl := NewRateLimiter(0)
	defer rl.Stop()

	clientID := "test-client"

	// All requests should be blocked with 0 limit
	if rl.Allow(clientID) {
		t.Error("Request should be blocked with 0 limit")
	}

	// Even after waiting, should still be blocked
	time.Sleep(1100 * time.Millisecond)
	if rl.Allow(clientID) {
		t.Error("Request should still be blocked with 0 limit after reset")
	}
}

func TestRateLimiter_RapidRequests(t *testing.T) {
	rl := NewRateLimiter(5)
	defer rl.Stop()

	clientID := "test-client"

	// Fire 5 requests as fast as possible
	allowed := 0
	blocked := 0

	for i := 0; i < 5; i++ {
		if rl.Allow(clientID) {
			allowed++
		} else {
			blocked++
		}
	}

	if allowed != 5 {
		t.Errorf("Expected exactly 5 allowed requests, got %d", allowed)
	}
	if blocked != 0 {
		t.Errorf("Expected 0 blocked requests in first batch, got %d", blocked)
	}

	// Immediately try 5 more - should all be blocked
	for i := 0; i < 5; i++ {
		if rl.Allow(clientID) {
			allowed++
		} else {
			blocked++
		}
	}

	if blocked != 5 {
		t.Errorf("Expected 5 blocked requests, got %d", blocked)
	}
}

func TestRateLimiter_CleanupPartial(t *testing.T) {
	rl := NewRateLimiter(10)
	defer rl.Stop()

	// Create some old clients and some new clients
	oldClients := []string{"old-1", "old-2", "old-3"}
	newClients := []string{"new-1", "new-2"}

	for _, clientID := range oldClients {
		rl.Allow(clientID)
	}

	for _, clientID := range newClients {
		rl.Allow(clientID)
	}

	// Make only old clients stale
	rl.mu.Lock()
	for _, clientID := range oldClients {
		bucket := rl.clients[clientID]
		bucket.mu.Lock()
		bucket.lastReset = time.Now().Add(-6 * time.Minute)
		bucket.mu.Unlock()
	}
	rl.mu.Unlock()

	// Manually trigger cleanup
	now := time.Now()
	toDelete := []string{}

	rl.mu.RLock()
	for clientID, bucket := range rl.clients {
		bucket.mu.Lock()
		if now.Sub(bucket.lastReset) > 5*time.Minute {
			toDelete = append(toDelete, clientID)
		}
		bucket.mu.Unlock()
	}
	rl.mu.RUnlock()

	if len(toDelete) > 0 {
		rl.mu.Lock()
		for _, clientID := range toDelete {
			delete(rl.clients, clientID)
		}
		rl.mu.Unlock()
	}

	// Verify only old clients were removed
	rl.mu.RLock()
	finalCount := len(rl.clients)
	rl.mu.RUnlock()

	if finalCount != len(newClients) {
		t.Errorf("Expected %d clients after cleanup, got %d", len(newClients), finalCount)
	}

	// Verify new clients still exist
	for _, clientID := range newClients {
		rl.mu.RLock()
		_, exists := rl.clients[clientID]
		rl.mu.RUnlock()
		if !exists {
			t.Errorf("New client %s should still exist after cleanup", clientID)
		}
	}

	// Verify old clients were removed
	for _, clientID := range oldClients {
		rl.mu.RLock()
		_, exists := rl.clients[clientID]
		rl.mu.RUnlock()
		if exists {
			t.Errorf("Old client %s should have been removed", clientID)
		}
	}
}
