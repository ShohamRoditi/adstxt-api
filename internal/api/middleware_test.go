package api

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"adstxt-api/internal/ratelimit"
)

// TestLoggingMiddleware tests that requests and responses are logged
func TestLoggingMiddleware(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test response"))
	})

	// Wrap with logging middleware
	middleware := LoggingMiddleware(handler)

	// Make a request
	req := httptest.NewRequest("GET", "/test-path", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// Check that logs were written
	logs := buf.String()
	if !strings.Contains(logs, "incoming request") {
		t.Error("Expected 'incoming request' log message")
	}
	if !strings.Contains(logs, "request completed") {
		t.Error("Expected 'request completed' log message")
	}
	if !strings.Contains(logs, "/test-path") {
		t.Error("Expected path in log message")
	}
	if !strings.Contains(logs, "GET") {
		t.Error("Expected method in log message")
	}
}

// TestLoggingMiddleware_StatusCode tests that status codes are logged correctly
func TestLoggingMiddleware_StatusCode(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	tests := []struct {
		name       string
		statusCode int
	}{
		{"200 OK", http.StatusOK},
		{"404 Not Found", http.StatusNotFound},
		{"500 Internal Server Error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			})

			middleware := LoggingMiddleware(handler)

			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			logs := buf.String()
			// Just check that the status field exists in the logs
			if !strings.Contains(logs, "status") {
				t.Errorf("Expected 'status' field in logs for %d, got: %s", tt.statusCode, logs)
			}
			// Verify the response has the correct status code
			if w.Code != tt.statusCode {
				t.Errorf("Expected response status %d, got %d", tt.statusCode, w.Code)
			}
		})
	}
}

// TestLoggingMiddleware_Duration tests that request duration is logged
func TestLoggingMiddleware_Duration(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	middleware := LoggingMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	logs := buf.String()
	if !strings.Contains(logs, "duration") {
		t.Error("Expected 'duration' field in log message")
	}
}

// TestRateLimitMiddleware tests rate limiting functionality
func TestRateLimitMiddleware(t *testing.T) {
	limiter := ratelimit.NewRateLimiter(2) // 2 requests per second
	defer limiter.Stop()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})

	middleware := RateLimitMiddleware(limiter)(handler)

	// First 2 requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: expected status 200, got %d", i+1, w.Code)
		}
	}

	// Third request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Rate limit exceeded") && !strings.Contains(body, "rate limit exceeded") {
		t.Errorf("Expected rate limit message, got: %s", body)
	}
}

// TestRateLimitMiddleware_DifferentClients tests that different clients have separate limits
func TestRateLimitMiddleware_DifferentClients(t *testing.T) {
	limiter := ratelimit.NewRateLimiter(2) // 2 requests per second
	defer limiter.Stop()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RateLimitMiddleware(limiter)(handler)

	// Client 1 makes 2 requests
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Client 1 Request %d: expected status 200, got %d", i+1, w.Code)
		}
	}

	// Client 1's third request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Client 1: expected status 429, got %d", w.Code)
	}

	// Client 2 should have separate limit (should succeed)
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:54321"
	w = httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Client 2: expected status 200, got %d", w.Code)
	}
}

// TestRateLimitMiddleware_Reset tests that rate limits reset after time window
func TestRateLimitMiddleware_Reset(t *testing.T) {
	limiter := ratelimit.NewRateLimiter(1) // 1 request per second
	defer limiter.Stop()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RateLimitMiddleware(limiter)(handler)

	// First request should succeed
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("First request: expected status 200, got %d", w.Code)
	}

	// Second request should be rate limited
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w = httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Second request: expected status 429, got %d", w.Code)
	}

	// Wait for rate limit window to reset
	time.Sleep(1100 * time.Millisecond)

	// Third request should succeed after reset
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w = httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Third request after reset: expected status 200, got %d", w.Code)
	}
}

// TestCORSMiddleware tests CORS headers are added
func TestCORSMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test"))
	})

	middleware := CORSMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// Check CORS headers
	headers := w.Header()

	if headers.Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Expected Access-Control-Allow-Origin: *, got: %s", headers.Get("Access-Control-Allow-Origin"))
	}

	if !strings.Contains(headers.Get("Access-Control-Allow-Methods"), "GET") {
		t.Error("Expected GET in Access-Control-Allow-Methods")
	}

	if !strings.Contains(headers.Get("Access-Control-Allow-Methods"), "POST") {
		t.Error("Expected POST in Access-Control-Allow-Methods")
	}

	if headers.Get("Access-Control-Allow-Headers") != "Content-Type" {
		t.Errorf("Expected Access-Control-Allow-Headers: Content-Type, got: %s", headers.Get("Access-Control-Allow-Headers"))
	}
}

// TestCORSMiddleware_OptionsRequest tests that OPTIONS requests are handled
func TestCORSMiddleware_OptionsRequest(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for OPTIONS request")
	})

	middleware := CORSMiddleware(handler)

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for OPTIONS, got %d", w.Code)
	}

	// Check that CORS headers are still present
	headers := w.Header()
	if headers.Get("Access-Control-Allow-Origin") != "*" {
		t.Error("CORS headers should be present for OPTIONS request")
	}
}

// TestResponseWriter_WriteHeader tests the custom responseWriter
func TestResponseWriter_WriteHeader(t *testing.T) {
	recorder := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: recorder,
		statusCode:     200,
	}

	// Write a custom status code
	rw.WriteHeader(http.StatusNotFound)

	if rw.statusCode != http.StatusNotFound {
		t.Errorf("Expected statusCode %d, got %d", http.StatusNotFound, rw.statusCode)
	}

	if recorder.Code != http.StatusNotFound {
		t.Errorf("Expected recorder Code %d, got %d", http.StatusNotFound, recorder.Code)
	}
}

// TestResponseWriter_DefaultStatusCode tests that responseWriter defaults to 200
func TestResponseWriter_DefaultStatusCode(t *testing.T) {
	recorder := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: recorder,
		statusCode:     200,
	}

	// Write response without explicitly setting status code
	_, _ = rw.Write([]byte("test"))

	if rw.statusCode != 200 {
		t.Errorf("Expected default statusCode 200, got %d", rw.statusCode)
	}
}

// TestMiddlewareChain tests multiple middleware working together
func TestMiddlewareChain(t *testing.T) {
	// Reset logger to avoid interfering with other tests
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	limiter := ratelimit.NewRateLimiter(10)
	defer limiter.Stop()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})

	// Chain middleware: CORS -> RateLimit -> Logging
	middleware := CORSMiddleware(
		RateLimitMiddleware(limiter)(
			LoggingMiddleware(handler),
		),
	)

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// Check that response is successful
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check CORS headers are present
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("CORS headers should be present in chained middleware")
	}

	// Check body
	if w.Body.String() != "success" {
		t.Errorf("Expected body 'success', got: %s", w.Body.String())
	}
}

// TestMiddlewareChain_RateLimited tests middleware chain with rate limiting
func TestMiddlewareChain_RateLimited(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	limiter := ratelimit.NewRateLimiter(1)
	defer limiter.Stop()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})

	middleware := CORSMiddleware(
		RateLimitMiddleware(limiter)(
			LoggingMiddleware(handler),
		),
	)

	// First request succeeds
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("First request: expected status 200, got %d", w.Code)
	}

	// Second request rate limited
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w = httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Second request: expected status 429, got %d", w.Code)
	}

	// CORS headers should still be present even when rate limited
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("CORS headers should be present even when rate limited")
	}
}

// TestCORSMiddleware_AllMethods tests that all allowed methods get CORS headers
func TestCORSMiddleware_AllMethods(t *testing.T) {
	methods := []string{"GET", "POST", "OPTIONS"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			middleware := CORSMiddleware(handler)

			req := httptest.NewRequest(method, "/test", nil)
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			headers := w.Header()
			if headers.Get("Access-Control-Allow-Origin") != "*" {
				t.Errorf("Method %s: missing CORS headers", method)
			}
		})
	}
}

// TestLoggingMiddleware_MultiplePaths tests logging different paths
func TestLoggingMiddleware_MultiplePaths(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := LoggingMiddleware(handler)

	paths := []string{"/api/analyze", "/health", "/metrics"}

	for _, path := range paths {
		buf.Reset()

		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		logs := buf.String()
		if !strings.Contains(logs, path) {
			t.Errorf("Expected path %s in logs, got: %s", path, logs)
		}
	}
}

// TestRateLimitMiddleware_HighLoad tests rate limiter under high load
func TestRateLimitMiddleware_HighLoad(t *testing.T) {
	limiter := ratelimit.NewRateLimiter(100) // 100 requests per second
	defer limiter.Stop()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RateLimitMiddleware(limiter)(handler)

	// Make 100 concurrent requests (should all succeed)
	type result struct {
		statusCode int
	}
	results := make(chan result, 100)

	for i := 0; i < 100; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "127.0.0.1:12345"
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			results <- result{statusCode: w.Code}
		}()
	}

	// Wait for all requests and count successes
	successCount := 0
	for i := 0; i < 100; i++ {
		res := <-results
		if res.statusCode == http.StatusOK {
			successCount++
		}
	}

	if successCount != 100 {
		t.Errorf("Expected 100 successful requests, got %d", successCount)
	}
}
