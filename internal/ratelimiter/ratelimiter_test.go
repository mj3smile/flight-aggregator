package ratelimiter

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestWait_ImmediateWhenAvailable(t *testing.T) {
	rl := NewTokenBucket(5, time.Second)

	start := time.Now()
	err := rl.Wait(context.Background())
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Wait() should succeed: %v", err)
	}
	if elapsed > 10*time.Millisecond {
		t.Errorf("expected immediate return, took %v", elapsed)
	}
}

func TestWait_BurstCapacity(t *testing.T) {
	rl := NewTokenBucket(5, time.Second)

	for i := range 5 {
		err := rl.Wait(context.Background())
		if err != nil {
			t.Fatalf("Wait() call %d should succeed: %v", i+1, err)
		}
	}
}

func TestWait_BlocksWhenExhausted(t *testing.T) {
	rl := NewTokenBucket(1, time.Second)

	err := rl.Wait(context.Background())
	if err != nil {
		t.Fatalf("first Wait() should succeed: %v", err)
	}

	start := time.Now()
	err = rl.Wait(context.Background())
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("second Wait() should succeed after refill: %v", err)
	}
	if elapsed < 1*time.Second {
		t.Errorf("expected to block for ~1s, but only took %v", elapsed)
	}
}

func TestWait_ContextCancelledWhileWaiting(t *testing.T) {
	rl := NewTokenBucket(1, time.Second)

	_ = rl.Wait(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := rl.Wait(ctx)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Wait() should return error when context is cancelled")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}
	if elapsed > 100*time.Millisecond {
		t.Errorf("expected prompt return on cancellation, took %v", elapsed)
	}
}

func TestWait_RefillsOverTime(t *testing.T) {
	rl := NewTokenBucket(2, time.Second) // refill 1 token every 500ms

	_ = rl.Wait(context.Background())
	_ = rl.Wait(context.Background())

	time.Sleep(600 * time.Millisecond)

	start := time.Now()
	err := rl.Wait(context.Background())
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Wait() after refill should succeed: %v", err)
	}
	if elapsed > 50*time.Millisecond {
		t.Errorf("expected immediate return after refill, took %v", elapsed)
	}
}

func TestWait_ConcurrentAccess(t *testing.T) {
	rl := NewTokenBucket(10, time.Second)

	var wg sync.WaitGroup
	successes := make(chan struct{}, 10)

	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			if rl.Wait(ctx) == nil {
				successes <- struct{}{}
			}
		}()
	}

	wg.Wait()
	close(successes)

	count := 0
	for range successes {
		count++
	}

	if count != 10 {
		t.Errorf("expected all 10 goroutines to get a token (bucket has 10), got %d", count)
	}
}

func TestAllow_WhenAvailable(t *testing.T) {
	rl := NewTokenBucket(5, time.Second)

	allowed, retryAfter := rl.Allow()
	if !allowed {
		t.Fatal("Allow() should succeed when tokens available")
	}
	if retryAfter != 0 {
		t.Errorf("retryAfter should be 0 when allowed, got %v", retryAfter)
	}
}

func TestAllow_WhenExhausted(t *testing.T) {
	rl := NewTokenBucket(1, time.Second)

	allowed, _ := rl.Allow()
	if !allowed {
		t.Fatal("first Allow() should succeed")
	}

	allowed, retryAfter := rl.Allow()
	if allowed {
		t.Fatal("second Allow() should be rejected")
	}
	if retryAfter <= 0 {
		t.Error("retryAfter should be positive when rejected")
	}
	if retryAfter > time.Second {
		t.Errorf("retryAfter should be at most 1s, got %v", retryAfter)
	}
}
