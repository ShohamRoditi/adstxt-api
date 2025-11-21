// =============================================================================
// internal/adstxt/parser.go
// =============================================================================

// =============================================================================
// internal/api/handler.go
// =============================================================================
package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"

	"adstxt-api/internal/adstxt"
	"adstxt-api/internal/cache"
	"adstxt-api/internal/config"
)

type Handler struct {
	cache   cache.Cache
	fetcher *adstxt.Fetcher
	cfg     *config.Config
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

func NewHandler(cache cache.Cache, cfg *config.Config) *Handler {
	return &Handler{
		cache:   cache,
		fetcher: adstxt.NewFetcher(cfg.RequestTimeout),
		cfg:     cfg,
	}
}

func (h *Handler) AnalyzeSingle(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Query().Get("domain")
	if domain == "" {
		h.sendError(w, http.StatusBadRequest, "domain parameter is required")
		return
	}

	log.Printf("Cache Type: %s", os.Getenv("CACHE_TYPE"))

	result, err := h.analyzeDomain(domain)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.sendJSON(w, http.StatusOK, result)
}

func (h *Handler) AnalyzeBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, http.StatusMethodNotAllowed, "only POST method is allowed")
		return
	}
	log.Printf("Cache Type: %s", os.Getenv("CACHE_TYPE"))

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
	h.sendJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func (h *Handler) analyzeDomain(domain string) (*SingleAnalysisResponse, error) {
	cacheKey := fmt.Sprintf("adstxt:%s", domain)
	cached := false

	cachedData, err := h.cache.Get(cacheKey)
	if err == nil {
		var result SingleAnalysisResponse
		if json.Unmarshal(cachedData, &result) == nil {
			result.Cached = true
			return &result, nil
		}
	}

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
		Cached:           cached,
		Timestamp:        time.Now().Format(time.RFC3339),
	}

	if data, err := json.Marshal(result); err == nil {
		h.cache.Set(cacheKey, data, 0)
	}

	return result, nil
}

func (h *Handler) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) sendError(w http.ResponseWriter, status int, message string) {
	h.sendJSON(w, status, ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
	})
}
