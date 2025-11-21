package api

import (
	"net/http"

	"adstxt-api/internal/ratelimit"
)

func NewRouter(handler *Handler, rateLimiter *ratelimit.RateLimiter) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", handler.Health)
	mux.HandleFunc("/api/analyze", handler.AnalyzeSingle)
	mux.HandleFunc("/api/batch-analysis", handler.AnalyzeBatch)

	var h http.Handler = mux
	h = CORSMiddleware(h)
	h = RateLimitMiddleware(rateLimiter)(h)
	h = LoggingMiddleware(h)

	return h
}
