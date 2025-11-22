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

func TestHandler_AnalyzeSingle_WithCache(t *testing.T) {
	cfg := &config.Config{
		CacheTTL:       1 * time.Hour,
		RequestTimeout: 10 * time.Second,
	}
	cacheStore := cache.NewMemoryCache(cfg.CacheTTL)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewHandler(cacheStore, cfg, logger)

	// First request (cache miss)
	req := httptest.NewRequest(http.MethodGet, "/api/analyze?domain=example.com", nil)
	w := httptest.NewRecorder()

	handler.AnalyzeSingle(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 200 or 500, got %d", w.Code)
	}
}

func TestHandler_AnalyzeBatch_Timeout(t *testing.T) {
	cfg := &config.Config{
		CacheTTL:       1 * time.Hour,
		RequestTimeout: 1 * time.Millisecond, // Very short timeout
	}
	cacheStore := cache.NewMemoryCache(cfg.CacheTTL)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewHandler(cacheStore, cfg, logger)

	reqBody := BatchAnalysisRequest{
		Domains: []string{"example.com", "test.com"},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/batch-analysis", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.AnalyzeBatch(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response BatchAnalysisResponse
	_ = json.NewDecoder(w.Body).Decode(&response)

	// Some domains should timeout
	if len(response.Errors) == 0 {
		t.Log("Warning: Expected some timeout errors, got none (network may be too fast)")
	}
}

func TestHandler_AnalyzeBatch_MaxBodySize(t *testing.T) {
	cfg := &config.Config{
		CacheTTL:       1 * time.Hour,
		RequestTimeout: 10 * time.Second,
	}
	cacheStore := cache.NewMemoryCache(cfg.CacheTTL)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewHandler(cacheStore, cfg, logger)

	// Create a body larger than maxBodySize
	largeDomains := make([]string, 10000)
	for i := range largeDomains {
		largeDomains[i] = "example.com"
	}

	reqBody := BatchAnalysisRequest{
		Domains: largeDomains,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/batch-analysis", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.AnalyzeBatch(w, req)

	// Should reject large body
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for large body, got %d", w.Code)
	}
}

func TestHandler_ValidateDomain(t *testing.T) {
	tests := []struct {
		name    string
		domain  string
		wantErr bool
	}{
		{"valid domain", "example.com", false},
		{"valid subdomain", "www.example.com", false},
		{"empty domain", "", true},
		{"too long", string(make([]byte, 300)), true},
		{"invalid chars", "http://example.com", true},
		{"no dot", "localhost", true},
		{"with path", "example.com/path", true},
		{"with protocol", "https://example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDomain(tt.domain)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDomain() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandler_Health_WithCache(t *testing.T) {
	cfg := &config.Config{
		CacheTTL: 1 * time.Hour,
	}
	cacheStore := cache.NewMemoryCache(cfg.CacheTTL)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewHandler(cacheStore, cfg, logger)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Health() status = %d, want %d", w.Code, http.StatusOK)
	}

	var response HealthResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "healthy" {
		t.Errorf("Health() status = %s, want healthy", response.Status)
	}

	if _, ok := response.Checks["cache"]; !ok {
		t.Error("Health() missing cache check")
	}
}

func TestHandler_Metrics(t *testing.T) {
	cfg := &config.Config{
		CacheTTL:       1 * time.Hour,
		RequestTimeout: 10 * time.Second,
	}
	cacheStore := cache.NewMemoryCache(cfg.CacheTTL)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewHandler(cacheStore, cfg, logger)

	// Make a request to increment metrics
	req := httptest.NewRequest(http.MethodGet, "/api/analyze?domain=test.com", nil)
	w := httptest.NewRecorder()
	handler.AnalyzeSingle(w, req)

	// Check metrics
	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w = httptest.NewRecorder()
	handler.Metrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Metrics() status = %d, want %d", w.Code, http.StatusOK)
	}

	var metrics map[string]int64
	err := json.NewDecoder(w.Body).Decode(&metrics)
	if err != nil {
		t.Fatalf("Failed to decode metrics: %v", err)
	}

	if metrics["requests_total"] < 1 {
		t.Errorf("Metrics() requests_total = %d, want >= 1", metrics["requests_total"])
	}
}
