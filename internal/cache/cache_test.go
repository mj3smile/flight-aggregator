package cache

import (
	"testing"
	"time"

	"github.com/mj3smile/flight-aggregator/internal/domain"
)

func TestCacheSetGet(t *testing.T) {
	c := NewInMemory(100)

	resp := &domain.SearchResponse{
		Metadata: domain.Metadata{TotalResults: 5},
	}
	c.Set("key1", resp, time.Minute)

	got := c.Get("key1")
	if got == nil {
		t.Fatal("expected cache hit")
	}
	if got.Metadata.TotalResults != 5 {
		t.Errorf("expected 5 results, got %d", got.Metadata.TotalResults)
	}
}

func TestCacheMiss(t *testing.T) {
	c := NewInMemory(100)

	got := c.Get("missing")
	if got != nil {
		t.Error("expected cache miss")
	}
}

func TestCacheTTLExpiry(t *testing.T) {
	c := NewInMemory(100)

	c.Set("key1", &domain.SearchResponse{}, 50*time.Millisecond)

	time.Sleep(100 * time.Millisecond)

	if c.Get("key1") != nil {
		t.Error("expected cache entry to expire")
	}
}

func TestCacheMaxSize(t *testing.T) {
	c := NewInMemory(2)

	c.Set("k1", &domain.SearchResponse{Metadata: domain.Metadata{TotalResults: 1}}, time.Minute)
	c.Set("k2", &domain.SearchResponse{Metadata: domain.Metadata{TotalResults: 2}}, time.Minute)
	c.Set("k3", &domain.SearchResponse{Metadata: domain.Metadata{TotalResults: 3}}, time.Minute)

	if c.Get("k3") == nil {
		t.Error("newest entry should exist")
	}

	nonNil := 0
	for _, k := range []string{"k1", "k2"} {
		if c.Get(k) != nil {
			nonNil++
		}
	}
	if nonNil > 1 {
		t.Error("at most one of k1/k2 should remain (max size 2)")
	}
}
