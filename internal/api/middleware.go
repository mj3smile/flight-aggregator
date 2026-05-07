package api

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/mj3smile/flight-aggregator/internal/ratelimiter"
)

type RateLimitMiddleware struct {
	mu       sync.Mutex
	limiters map[string]ratelimiter.RateLimiter
	burst    int
	interval time.Duration
}

func NewRateLimitMiddleware(burst int, interval time.Duration) *RateLimitMiddleware {
	rl := &RateLimitMiddleware{
		limiters: make(map[string]ratelimiter.RateLimiter),
		burst:    burst,
		interval: interval,
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimitMiddleware) getLimiter(ip string) ratelimiter.RateLimiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	lim, ok := rl.limiters[ip]
	if !ok {
		lim = ratelimiter.NewTokenBucket(rl.burst, rl.interval)
		rl.limiters[ip] = lim
	}
	return lim
}

func (rl *RateLimitMiddleware) getClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (rl *RateLimitMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := rl.getClientIP(r)
		limiter := rl.getLimiter(ip)

		allowed, retryAfter := limiter.Allow()
		if !allowed {
			w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
			writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimitMiddleware) cleanup() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		for ip, lim := range rl.limiters {
			if time.Since(lim.LastRefill()) >= rl.interval {
				delete(rl.limiters, ip)
			}
		}
		rl.mu.Unlock()
	}
}
