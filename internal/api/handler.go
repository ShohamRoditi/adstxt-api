package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"adstxt-api/internal/adstxt"
	"adstxt-api/internal/cache"
	"adstxt-api/internal/config"
)

const maxBodySize = 1 << 20 // 1MB

type Handler struct {
	cache   cache.Cache
	fetcher *adstxt.Fetcher
	cfg     *config.Config
	logger  *slog.Logger
	metrics *Metrics
}

type SingleAnalysisResponse struct {
	Domain           string                   `json:"domain"`
	TotalAdvertisers int                      `json:"total_advertisers"`
	Advertisers      []adstxt.AdvertiserCount `json:"advertisers"`
	Cached           bool                     `json:"cached"`
	Timestamp        string                   `json:"timestamp"`
}

type BatchAnalysisRequest struct {
	Domains []string `json:"domains"`
}

type BatchAnalysisResponse struct {
	Results []SingleAnalysisResponse `json:"results"`
	Errors  map[string]string        `json:"errors,omitempty"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

type HealthResponse struct {
	Status  string            `json:"status"`
	Time    string            `json:"time"`
	Version string            `json:"version,omitempty"`
	Checks  map[string]string `json:"checks"`
}

type Metrics struct {
	requestsTotal int64
	cacheHits     int64
	cacheMisses   int64
	errorTotal    int64
	mu            sync.RWMutex
}

func NewHandler(cache cache.Cache, cfg *config.Config, logger *slog.Logger) *Handler {
	return &Handler{
		cache:   cache,
		fetcher: adstxt.NewFetcher(cfg.RequestTimeout),
		cfg:     cfg,
		logger:  logger,
		metrics: &Metrics{},
	}
}

func validateDomain(domain string) error {
	if domain == "" {
		return errors.New("domain cannot be empty")
	}

	// Check length
	if len(domain) > 253 {
		return errors.New("domain too long")
	}

	// Check for invalid characters
	if strings.ContainsAny(domain, "/:@?#[]!$&'()*+,;= ") {
		return errors.New("invalid domain format")
	}

	// Check basic format
	if !strings.Contains(domain, ".") {
		return errors.New("invalid domain format")
	}

	return nil
}

func (h *Handler) AnalyzeSingle(w http.ResponseWriter, r *http.Request) {
	h.metrics.mu.Lock()
	h.metrics.requestsTotal++
	h.metrics.mu.Unlock()

	domain := r.URL.Query().Get("domain")
	if err := validateDomain(domain); err != nil {
		h.logger.Warn("invalid domain", slog.String("domain", domain), slog.String("error", err.Error()))
		h.sendError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.logger.Info("analyzing domain", slog.String("domain", domain))
	result, err := h.analyzeDomain(domain)
	if err != nil {
		h.metrics.mu.Lock()
		h.metrics.errorTotal++
		h.metrics.mu.Unlock()
		h.logger.Error("failed to analyze domain", slog.String("domain", domain), slog.String("error", err.Error()))
		h.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.logger.Info("domain analyzed successfully",
		slog.String("domain", domain),
		slog.Bool("cached", result.Cached),
		slog.Int("advertisers", result.TotalAdvertisers))
	h.sendJSON(w, http.StatusOK, result)
}

func (h *Handler) AnalyzeBatch(w http.ResponseWriter, r *http.Request) {
	h.metrics.mu.Lock()
	h.metrics.requestsTotal++
	h.metrics.mu.Unlock()

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if r.Method != http.MethodPost {
		h.sendError(w, http.StatusMethodNotAllowed, "only POST method is allowed")
		return
	}
	// Limit body size
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	var req BatchAnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	if len(req.Domains) == 0 {
		h.sendError(w, http.StatusBadRequest, "domains array cannot be empty")
		return
	}

	if len(req.Domains) > 50 {
		h.sendError(w, http.StatusBadRequest, "maximum 50 domains per batch request")
		return
	}

	response := BatchAnalysisResponse{
		Results: make([]SingleAnalysisResponse, 0),
		Errors:  make(map[string]string),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, domain := range req.Domains {
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			// Check context cancellation
			select {
			case <-ctx.Done():
				mu.Lock()
				response.Errors[d] = "request timeout"
				mu.Unlock()
				return
			default:
			}

			// Validate domain to prevent SSRF attacks
			if err := validateDomain(d); err != nil {
				mu.Lock()
				response.Errors[d] = "invalid domain: " + err.Error()
				mu.Unlock()
				return
			}

			result, err := h.analyzeDomain(d)
			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				response.Errors[d] = err.Error()
			} else {
				response.Results = append(response.Results, *result)
			}
		}(domain)
	}

	wg.Wait()

	h.sendJSON(w, http.StatusOK, response)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	checks := make(map[string]string)
	overallStatus := "healthy"

	// Check cache
	testKey := "health:check"
	if err := h.cache.Set(testKey, []byte("ok"), 10*time.Second); err != nil {
		checks["cache"] = "unhealthy: " + err.Error()
		overallStatus = "degraded"
	} else {
		checks["cache"] = "healthy"
		if err := h.cache.Delete(testKey); err != nil {
			h.logger.Warn("failed to delete health check key", slog.String("error", err.Error()))
		}
	}

	response := HealthResponse{
		Status:  overallStatus,
		Time:    time.Now().Format(time.RFC3339),
		Version: "1.0.0",
		Checks:  checks,
	}

	statusCode := http.StatusOK
	if overallStatus != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}

	h.sendJSON(w, statusCode, response)
}

func (h *Handler) Metrics(w http.ResponseWriter, r *http.Request) {
	h.metrics.mu.RLock()
	defer h.metrics.mu.RUnlock()

	h.sendJSON(w, http.StatusOK, map[string]int64{
		"requests_total": h.metrics.requestsTotal,
		"cache_hits":     h.metrics.cacheHits,
		"cache_misses":   h.metrics.cacheMisses,
		"errors_total":   h.metrics.errorTotal,
	})
}

func (h *Handler) analyzeDomain(domain string) (*SingleAnalysisResponse, error) {
	cacheKey := fmt.Sprintf("adstxt:%s", domain)

	// Try to get from cache (works for all cache types: memory, file, redis)
	cachedData, err := h.cache.Get(cacheKey)
	if err == nil {
		var result SingleAnalysisResponse
		if unmarshalErr := json.Unmarshal(cachedData, &result); unmarshalErr == nil {
			result.Cached = true
			h.metrics.mu.Lock()
			h.metrics.cacheHits++
			h.metrics.mu.Unlock()
			return &result, nil
		} else {
			h.logger.Warn("failed to unmarshal cached data",
				slog.String("domain", domain),
				slog.String("error", unmarshalErr.Error()))
		}
	}

	// Cache miss - fetch fresh data
	h.metrics.mu.Lock()
	h.metrics.cacheMisses++
	h.metrics.mu.Unlock()

	content, err := h.fetcher.FetchAdsTxt(domain)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ads.txt: %v", err)
	}

	advertisersMap := adstxt.ParseAdsTxt(content)
	advertisers := adstxt.MapToSlice(advertisersMap)

	sort.Slice(advertisers, func(i, j int) bool {
		if advertisers[i].Count == advertisers[j].Count {
			return advertisers[i].Domain < advertisers[j].Domain
		}
		return advertisers[i].Count > advertisers[j].Count
	})

	result := &SingleAnalysisResponse{
		Domain:           domain,
		TotalAdvertisers: len(advertisers),
		Advertisers:      advertisers,
		Cached:           false, // Fresh data, not from cache
		Timestamp:        time.Now().Format(time.RFC3339),
	}

	// Store in cache for future requests (works for all cache types)
	if data, err := json.Marshal(result); err == nil {
		if err := h.cache.Set(cacheKey, data, h.cfg.CacheTTL); err != nil {
			h.logger.Warn("failed to cache result", slog.String("domain", domain), slog.String("error", err.Error()))
		}
	}

	return result, nil
}

func (h *Handler) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	defer func() {
		if r := recover(); r != nil {
			h.logger.Error("panic in JSON encoding", slog.Any("panic", r))
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode JSON response", slog.String("error", err.Error()))
	}
}

func (h *Handler) sendError(w http.ResponseWriter, status int, message string) {
	h.sendJSON(w, status, ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
	})
}
