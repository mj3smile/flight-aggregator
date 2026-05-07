package aggregator

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/mj3smile/flight-aggregator/internal/cache"
	"github.com/mj3smile/flight-aggregator/internal/domain"
	"github.com/mj3smile/flight-aggregator/internal/provider"
	"github.com/mj3smile/flight-aggregator/internal/ratelimiter"
	"github.com/mj3smile/flight-aggregator/internal/search"
)

type Config struct {
	ProviderTimeout time.Duration
	MaxRetries      int
	CacheTTL        time.Duration
}

func DefaultConfig() Config {
	return Config{
		ProviderTimeout: 2 * time.Second,
		MaxRetries:      3,
		CacheTTL:        5 * time.Minute,
	}
}

type Aggregator struct {
	providers    []provider.Provider
	cache        cache.Cache
	config       Config
	rateLimiters map[string]ratelimiter.RateLimiter
}

func New(cfg Config, providers []provider.Provider, c cache.Cache, rls map[string]ratelimiter.RateLimiter) *Aggregator {
	return &Aggregator{
		providers:    providers,
		cache:        c,
		config:       cfg,
		rateLimiters: rls,
	}
}

func (a *Aggregator) Search(ctx context.Context, req domain.SearchRequest) (*domain.SearchResponse, error) {
	key := domain.CacheKey(req)
	if cached := a.cache.Get(key); cached != nil {
		result := *cached
		result.Metadata.CacheHit = true
		if req.Filters != nil || req.SortBy != "" {
			result.Flights = search.Apply(result.Flights, req.Filters, req.SortBy)
			result.Metadata.TotalResults = len(result.Flights)
		}
		return &result, nil
	}

	start := time.Now()
	type providerResult struct {
		name    string
		flights []domain.Flight
		err     error
	}

	results := make(chan providerResult, len(a.providers))
	var wg sync.WaitGroup

	for _, p := range a.providers {
		wg.Add(1)
		go func(p provider.Provider) {
			defer wg.Done()
			flights, err := a.queryWithRetry(ctx, p, req)
			results <- providerResult{name: p.Name(), flights: flights, err: err}
		}(p)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var allFlights []domain.Flight
	succeeded := 0
	failed := 0

	for res := range results {
		if res.err != nil {
			slog.Warn("provider failed", "provider", res.name, "error", res.err)
			failed++
			continue
		}
		succeeded++
		allFlights = append(allFlights, res.flights...)
	}

	if succeeded == 0 {
		return nil, fmt.Errorf("all %d providers failed", failed)
	}

	allFlights = validateFlights(allFlights)
	allFlights = filterByRequest(allFlights, req)
	elapsed := time.Since(start).Milliseconds()
	criteria := domain.SearchCriteria{
		Origin:        req.Origin,
		Destination:   req.Destination,
		DepartureDate: req.DepartureDate,
		Passengers:    req.Passengers,
		CabinClass:    req.CabinClass,
	}

	meta := domain.Metadata{
		TotalResults:       len(allFlights),
		ProvidersQueried:   len(a.providers),
		ProvidersSucceeded: succeeded,
		ProvidersFailed:    failed,
		SearchTimeMs:       elapsed,
		CacheHit:           false,
	}

	if failed == 0 {
		a.cache.Set(key, &domain.SearchResponse{
			SearchCriteria: criteria,
			Metadata:       meta,
			Flights:        allFlights,
		}, a.config.CacheTTL)
	}

	filtered := search.Apply(allFlights, req.Filters, req.SortBy)
	meta.TotalResults = len(filtered)
	return &domain.SearchResponse{
		SearchCriteria: criteria,
		Metadata:       meta,
		Flights:        filtered,
	}, nil
}

func (a *Aggregator) queryWithRetry(ctx context.Context, p provider.Provider, req domain.SearchRequest) ([]domain.Flight, error) {
	var lastErr error
	rl := a.rateLimiters[p.Name()]
	for attempt := range a.config.MaxRetries {
		if rl != nil {
			if err := rl.Wait(ctx); err != nil {
				slog.Warn("rate limiter failed, proceeding anyway", "provider", p.Name(), "error", err)
			}
		}

		pCtx, cancel := context.WithTimeout(ctx, a.config.ProviderTimeout)
		flights, err := p.Search(pCtx, req)
		cancel()

		if err == nil {
			return flights, nil
		}
		lastErr = err

		if attempt == a.config.MaxRetries-1 {
			break
		}

		backoff := time.Duration(1<<uint(attempt)) * 500 * time.Millisecond
		slog.Debug("provider retry", "provider", p.Name(), "attempt", attempt+1, "backoff", backoff)
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return nil, fmt.Errorf("all %d retries failed: %w", a.config.MaxRetries, lastErr)
}

func filterByRequest(flights []domain.Flight, req domain.SearchRequest) []domain.Flight {
	result := make([]domain.Flight, 0, len(flights))
	for _, f := range flights {
		depDate := f.Departure.Datetime[:10]
		if depDate != req.DepartureDate {
			continue
		}
		if f.Departure.Airport != req.Origin {
			continue
		}
		if f.Arrival.Airport != req.Destination {
			continue
		}
		if req.Passengers > 0 && f.AvailableSeats < req.Passengers {
			continue
		}
		if req.CabinClass != "" && f.CabinClass != req.CabinClass {
			continue
		}
		result = append(result, f)
	}
	return result
}

func validateFlights(flights []domain.Flight) []domain.Flight {
	valid := make([]domain.Flight, 0, len(flights))
	for _, f := range flights {
		if f.Departure.Timestamp >= f.Arrival.Timestamp {
			slog.Warn("invalid flight: arrival not after departure", "flight", f.ID)
			continue
		}
		if f.Duration.TotalMinutes <= 0 {
			slog.Warn("invalid flight: non-positive duration", "flight", f.ID)
			continue
		}
		if f.Price.Amount <= 0 {
			slog.Warn("invalid flight: non-positive price", "flight", f.ID)
			continue
		}
		valid = append(valid, f)
	}
	return valid
}
