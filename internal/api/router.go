package api

import (
	"net/http"

	"adstxt-api/internal/ratelimit"
)

// NewRouter creates and configures the HTTP router with all endpoints and middleware.
// It sets up the following routes:
//   - GET  /health          - Health check endpoint
//   - GET  /metrics         - Metrics endpoint
//   - GET  /api/analyze     - Single domain analysis (with ?domain= query param)
//   - POST /api/batch-analysis - Batch domain analysis
//
// The router applies middleware in the following order:
//  1. LoggingMiddleware    - Logs all requests and responses
//  2. RateLimitMiddleware  - Rate limiting per client IP
//  3. CORSMiddleware       - CORS headers for cross-origin requests
func NewRouter(handler *Handler, rateLimiter *ratelimit.RateLimiter) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", handler.Health)
	mux.HandleFunc("/metrics", handler.Metrics)
	mux.HandleFunc("/api/analyze", handler.AnalyzeSingle)
	mux.HandleFunc("/api/batch-analysis", handler.AnalyzeBatch)

	var h http.Handler = mux
	h = CORSMiddleware(h)
	h = RateLimitMiddleware(rateLimiter)(h)
	h = LoggingMiddleware(h)

	return h
}
