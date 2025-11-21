package main

import (
	"context"
	"log"
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
	cfg := config.Load()

	cacheStore, err := cache.NewCache(cfg.CacheType, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize cache: %v", err)
	}
	defer cacheStore.Close()

	rateLimiter := ratelimit.NewRateLimiter(cfg.RateLimitPerSecond)

	handler := api.NewHandler(cacheStore, cfg)
	router := api.NewRouter(handler, rateLimiter)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Server starting on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
