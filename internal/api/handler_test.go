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
