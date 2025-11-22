package api

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"adstxt-api/internal/ratelimit"
)

// responseWriter wraps http.ResponseWriter to capture the status code for logging.
// This allows middleware to log the response status without interfering with the handler.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code and delegates to the underlying ResponseWriter.
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// LoggingMiddleware logs all HTTP requests and responses with structured logging.
// It logs the request method, path, and remote address when the request starts,
// and logs the status code and duration when the request completes.
// Uses slog for structured JSON logging with contextual fields.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}

		slog.Info("incoming request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("remote_addr", r.RemoteAddr))

		next.ServeHTTP(wrapped, r)

		slog.Info("request completed",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", wrapped.statusCode),
			slog.Duration("duration", time.Since(start)))
	})
}

// RateLimitMiddleware creates a middleware that enforces rate limiting per client IP.
// It uses the provided RateLimiter to track and limit requests from each remote address.
// If a client exceeds the rate limit, a 429 Too Many Requests response is returned.
// The middleware extracts only the IP address from r.RemoteAddr (strips the port).
func RateLimitMiddleware(limiter *ratelimit.RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract IP without port (r.RemoteAddr format: "IP:port")
			clientIP := r.RemoteAddr
			if colonIndex := strings.LastIndex(clientIP, ":"); colonIndex != -1 {
				clientIP = clientIP[:colonIndex]
			}

			if !limiter.Allow(clientIP) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"Rate limit exceeded","message":"Too many requests. Please try again later."}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CORSMiddleware adds Cross-Origin Resource Sharing (CORS) headers to all responses.
// It allows requests from any origin (*) with common HTTP methods and headers.
// Pre-flight OPTIONS requests are handled automatically and return 200 OK.
// This enables the API to be called from web browsers running on different domains.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
