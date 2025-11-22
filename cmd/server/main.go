package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"adstxt-api/internal/api"
	"adstxt-api/internal/cache"
	"adstxt-api/internal/config"
	"adstxt-api/internal/ratelimit"
)

func main() {
	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg := config.Load()

	logger.Info("loading configuration",
		slog.String("port", cfg.Port),
		slog.String("cache_type", cfg.CacheType),
		slog.Duration("cache_ttl", cfg.CacheTTL),
		slog.Int("rate_limit", cfg.RateLimitPerSecond),
	)

	cacheStore, err := cache.NewCache(cfg.CacheType, cfg)
	if err != nil {
		logger.Error("failed to initialize cache", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer cacheStore.Close()

	rateLimiter := ratelimit.NewRateLimiter(cfg.RateLimitPerSecond)
	defer rateLimiter.Stop()

	handler := api.NewHandler(cacheStore, cfg, logger)
	router := api.NewRouter(handler, rateLimiter)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("server starting", slog.String("port", cfg.Port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("server exited successfully")
}
