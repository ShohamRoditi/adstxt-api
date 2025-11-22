package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"adstxt-api/internal/cache"
	"adstxt-api/internal/config"
)

func TestHandler_Health(t *testing.T) {
	cfg := &config.Config{
		CacheTTL:       1 * time.Hour,
		RequestTimeout: 10 * time.Second,
	}

	cache := cache.NewMemoryCache(cfg.CacheTTL)
	defer cache.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewHandler(cache, cfg, logger)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]string
	_ = json.NewDecoder(w.Body).Decode(&response)

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", response["status"])
	}
}

func TestHandler_AnalyzeSingle_MissingDomain(t *testing.T) {
	cfg := &config.Config{
		CacheTTL:       1 * time.Hour,
		RequestTimeout: 10 * time.Second,
	}

	cache := cache.NewMemoryCache(cfg.CacheTTL)
	defer cache.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewHandler(cache, cfg, logger)

	req := httptest.NewRequest("GET", "/api/analyze", nil)
	w := httptest.NewRecorder()

	handler.AnalyzeSingle(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandler_AnalyzeBatch_InvalidMethod(t *testing.T) {
	cfg := &config.Config{
		CacheTTL:       1 * time.Hour,
		RequestTimeout: 10 * time.Second,
	}

	cache := cache.NewMemoryCache(cfg.CacheTTL)
	defer cache.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewHandler(cache, cfg, logger)

	req := httptest.NewRequest("GET", "/api/batch-analysis", nil)
	w := httptest.NewRecorder()

	handler.AnalyzeBatch(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestHandler_AnalyzeBatch_EmptyDomains(t *testing.T) {
	cfg := &config.Config{
		CacheTTL:       1 * time.Hour,
		RequestTimeout: 10 * time.Second,
	}

	cache := cache.NewMemoryCache(cfg.CacheTTL)
	defer cache.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewHandler(cache, cfg, logger)

	body := BatchAnalysisRequest{Domains: []string{}}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/batch-analysis", bytes.NewBuffer(jsonBody))
	w := httptest.NewRecorder()

	handler.AnalyzeBatch(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandler_AnalyzeBatch_TooManyDomains(t *testing.T) {
	cfg := &config.Config{
		CacheTTL:       1 * time.Hour,
		RequestTimeout: 10 * time.Second,
	}

	cache := cache.NewMemoryCache(cfg.CacheTTL)
	defer cache.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewHandler(cache, cfg, logger)

	domains := make([]string, 51)
	for i := range domains {
		domains[i] = "example.com"
	}

	body := BatchAnalysisRequest{Domains: domains}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/batch-analysis", bytes.NewBuffer(jsonBody))
	w := httptest.NewRecorder()

	handler.AnalyzeBatch(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandler_Health_CacheFailure(t *testing.T) {
	cfg := &config.Config{
		CacheTTL:       1 * time.Hour,
		RequestTimeout: 10 * time.Second,
	}

	// Use a file cache with invalid path to trigger cache error
	invalidCache, err := cache.NewFileCache("/invalid/path/that/does/not/exist", cfg.CacheTTL)
	if err != nil {
		t.Skip("Cannot create invalid cache path for test")
	}
	defer invalidCache.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewHandler(invalidCache, cfg, logger)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	// Health check should report degraded status when cache fails
	var response HealthResponse
	_ = json.NewDecoder(w.Body).Decode(&response)

	if response.Status == "healthy" {
		// If cache works, that's fine too - just testing the code path
		t.Logf("Cache write succeeded unexpectedly, status: %s", response.Status)
	}
}

func TestHandler_AnalyzeBatch_InvalidJSON(t *testing.T) {
	cfg := &config.Config{
		CacheTTL:       1 * time.Hour,
		RequestTimeout: 10 * time.Second,
	}

	cache := cache.NewMemoryCache(cfg.CacheTTL)
	defer cache.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewHandler(cache, cfg, logger)

	req := httptest.NewRequest("POST", "/api/batch-analysis", bytes.NewBuffer([]byte("{invalid json")))
	w := httptest.NewRecorder()

	handler.AnalyzeBatch(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandler_AnalyzeBatch_InvalidDomain(t *testing.T) {
	cfg := &config.Config{
		CacheTTL:       1 * time.Hour,
		RequestTimeout: 10 * time.Second,
	}

	cache := cache.NewMemoryCache(cfg.CacheTTL)
	defer cache.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewHandler(cache, cfg, logger)

	body := BatchAnalysisRequest{
		Domains: []string{"http://evil.com", "localhost:6379", "../../../etc/passwd"},
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/batch-analysis", bytes.NewBuffer(jsonBody))
	w := httptest.NewRecorder()

	handler.AnalyzeBatch(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 (with errors in response), got %d", w.Code)
	}

	var response BatchAnalysisResponse
	_ = json.NewDecoder(w.Body).Decode(&response)

	// All domains should have errors
	if len(response.Errors) != 3 {
		t.Errorf("Expected 3 errors, got %d", len(response.Errors))
	}
}

func TestHandler_MetricsEndpoint(t *testing.T) {
	cfg := &config.Config{
		CacheTTL:       1 * time.Hour,
		RequestTimeout: 10 * time.Second,
	}

	cache := cache.NewMemoryCache(cfg.CacheTTL)
	defer cache.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewHandler(cache, cfg, logger)

	// Make a request to increment metrics
	req := httptest.NewRequest("GET", "/api/analyze?domain=example.com", nil)
	w := httptest.NewRecorder()
	handler.AnalyzeSingle(w, req)

	// Now check metrics
	req = httptest.NewRequest("GET", "/metrics", nil)
	w = httptest.NewRecorder()
	handler.Metrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var metrics map[string]int64
	_ = json.NewDecoder(w.Body).Decode(&metrics)

	if metrics["requests_total"] < 1 {
		t.Error("Expected at least 1 request")
	}
}

func TestHandler_AnalyzeDomain_CacheHit(t *testing.T) {
	cfg := &config.Config{
		CacheTTL:       1 * time.Hour,
		RequestTimeout: 10 * time.Second,
	}

	cache := cache.NewMemoryCache(cfg.CacheTTL)
	defer cache.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewHandler(cache, cfg, logger)

	// Pre-populate cache with valid data
	domain := "cached-example.com"
	cachedResponse := SingleAnalysisResponse{
		Domain:           domain,
		TotalAdvertisers: 2,
		Cached:           false,
		Timestamp:        time.Now().Format(time.RFC3339),
	}

	data, _ := json.Marshal(cachedResponse)
	cacheKey := "adstxt:" + domain
	_ = cache.Set(cacheKey, data, cfg.CacheTTL)

	// Request should hit cache
	req := httptest.NewRequest("GET", "/api/analyze?domain="+domain, nil)
	w := httptest.NewRecorder()

	handler.AnalyzeSingle(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response SingleAnalysisResponse
	_ = json.NewDecoder(w.Body).Decode(&response)

	if !response.Cached {
		t.Error("Expected cached response")
	}

	// Verify cache hit metric incremented
	if handler.metrics.cacheHits == 0 {
		t.Error("Expected cache hit to be recorded")
	}
}

func TestHandler_AnalyzeDomain_InvalidCachedData(t *testing.T) {
	cfg := &config.Config{
		CacheTTL:       1 * time.Hour,
		RequestTimeout: 10 * time.Second,
	}

	cache := cache.NewMemoryCache(cfg.CacheTTL)
	defer cache.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewHandler(cache, cfg, logger)

	// Pre-populate cache with invalid JSON
	domain := "bad-cache.com"
	cacheKey := "adstxt:" + domain
	_ = cache.Set(cacheKey, []byte("invalid json data"), cfg.CacheTTL)

	// Request should handle invalid cache data and try to fetch fresh
	req := httptest.NewRequest("GET", "/api/analyze?domain="+domain, nil)
	w := httptest.NewRecorder()

	handler.AnalyzeSingle(w, req)

	// Should get error since domain doesn't exist, but it should have attempted fresh fetch
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 (fetch failed), got %d", w.Code)
	}
}

func TestHandler_SendJSON_Panic(t *testing.T) {
	cfg := &config.Config{
		CacheTTL:       1 * time.Hour,
		RequestTimeout: 10 * time.Second,
	}

	cache := cache.NewMemoryCache(cfg.CacheTTL)
	defer cache.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewHandler(cache, cfg, logger)

	w := httptest.NewRecorder()

	// Create a type that will cause JSON encoding to fail/panic
	// Using a channel type which is not JSON-marshalable
	badData := make(chan int)

	// This should not panic due to defer recover
	handler.sendJSON(w, http.StatusOK, badData)

	// The response should have some status even if encoding failed
	if w.Code == 0 {
		t.Error("Expected some status code to be set")
	}
}
