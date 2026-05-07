package ratelimiter

import (
	"context"
	"sync"
	"time"
)

type RateLimiter interface {
	Wait(ctx context.Context) error
	Allow() (bool, time.Duration)
	LastRefill() time.Time
}

type tokenBucketLimiter struct {
	mu         sync.Mutex
	tokens     float64
	lastRefill time.Time
	maxTokens  float64
	rate       float64
}

func NewTokenBucket(maxPerInterval int, interval time.Duration) RateLimiter {
	return &tokenBucketLimiter{
		tokens:     float64(maxPerInterval),
		lastRefill: time.Now(),
		maxTokens:  float64(maxPerInterval),
		rate:       float64(maxPerInterval) / interval.Seconds(),
	}
}

func (rl *tokenBucketLimiter) tryConsume() (bool, time.Duration) {
	now := time.Now()

	rl.tokens += now.Sub(rl.lastRefill).Seconds() * rl.rate
	if rl.tokens > rl.maxTokens {
		rl.tokens = rl.maxTokens
	}
	rl.lastRefill = now

	if rl.tokens >= 1 {
		rl.tokens--
		return true, 0
	}

	retryAfter := time.Duration((1 - rl.tokens) / rl.rate * float64(time.Second))
	return false, retryAfter
}

func (rl *tokenBucketLimiter) LastRefill() time.Time {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.lastRefill
}

func (rl *tokenBucketLimiter) Allow() (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.tryConsume()
}

func (rl *tokenBucketLimiter) Wait(ctx context.Context) error {
	rl.mu.Lock()
	allowed, retryAfter := rl.tryConsume()
	if allowed {
		rl.mu.Unlock()
		return nil
	}

	rl.tokens = 0
	rl.lastRefill = time.Now().Add(retryAfter)
	rl.mu.Unlock()

	select {
	case <-time.After(retryAfter):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
