package cache

import (
	"sync"
	"time"

	"github.com/mj3smile/flight-aggregator/internal/domain"
)

type Cache interface {
	Get(key string) *domain.SearchResponse
	Set(key string, resp *domain.SearchResponse, ttl time.Duration)
}

type entry struct {
	response  *domain.SearchResponse
	expiresAt time.Time
}

type InMemory struct {
	mu      sync.RWMutex
	items   map[string]entry
	maxSize int
}

func NewInMemory(maxSize int) *InMemory {
	c := &InMemory{
		items:   make(map[string]entry),
		maxSize: maxSize,
	}
	go c.cleanup()
	return c
}

func (c *InMemory) Get(key string) *domain.SearchResponse {
	c.mu.RLock()
	defer c.mu.RUnlock()

	e, ok := c.items[key]
	if !ok || time.Now().After(e.expiresAt) {
		return nil
	}
	return e.response
}

func (c *InMemory) Set(key string, resp *domain.SearchResponse, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.items) >= c.maxSize {
		c.evictOldest()
	}

	c.items[key] = entry{
		response:  resp,
		expiresAt: time.Now().Add(ttl),
	}
}

func (c *InMemory) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	first := true

	for k, v := range c.items {
		if first || v.expiresAt.Before(oldestTime) {
			oldestKey = k
			oldestTime = v.expiresAt
			first = false
		}
	}

	if oldestKey != "" {
		delete(c.items, oldestKey)
	}
}

func (c *InMemory) cleanup() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for k, v := range c.items {
			if now.After(v.expiresAt) {
				delete(c.items, k)
			}
		}
		c.mu.Unlock()
	}
}
