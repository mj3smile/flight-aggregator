package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mj3smile/flight-aggregator/internal/aggregator"
	"github.com/mj3smile/flight-aggregator/internal/api"
	"github.com/mj3smile/flight-aggregator/internal/cache"
	"github.com/mj3smile/flight-aggregator/internal/provider"
	"github.com/mj3smile/flight-aggregator/internal/ratelimiter"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	providers := []provider.Provider{
		provider.NewGaruda(),
		provider.NewLionAir(),
		provider.NewBatikAir(),
		provider.NewAirAsia(),
	}

	c := cache.NewInMemory(1000)
	rls := buildRateLimiters(providers)

	cfg := aggregator.DefaultConfig()
	agg := aggregator.New(cfg, providers, c, rls)

	handler := api.NewHandler(agg)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	rateLimiter := api.NewRateLimitMiddleware(30, time.Minute)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      rateLimiter.Wrap(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server starting", "port", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

func buildRateLimiters(providers []provider.Provider) map[string]ratelimiter.RateLimiter {
	rls := make(map[string]ratelimiter.RateLimiter, len(providers))
	for _, p := range providers {
		cfg := p.RateLimit()
		if cfg == nil {
			continue
		}
		rls[p.Name()] = ratelimiter.NewTokenBucket(cfg.Burst, cfg.Interval)
	}
	return rls
}
