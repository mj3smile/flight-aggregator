package aggregator

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/mj3smile/flight-aggregator/internal/cache"
	"github.com/mj3smile/flight-aggregator/internal/domain"
	"github.com/mj3smile/flight-aggregator/internal/mocks"
	"github.com/mj3smile/flight-aggregator/internal/provider"
	"github.com/mj3smile/flight-aggregator/internal/ratelimiter"
	"go.uber.org/mock/gomock"
)

var testReq = domain.SearchRequest{
	Origin:        "CGK",
	Destination:   "DPS",
	DepartureDate: "2025-12-15",
	Passengers:    1,
	CabinClass:    domain.CabinEconomy,
}

func validFlight() domain.Flight {
	return domain.Flight{
		ID:             "TEST1_Mock",
		Provider:       "Mock",
		Airline:        domain.Airline{Name: "Mock Air", Code: "MK"},
		FlightNumber:   "TEST1",
		Departure:      domain.FlightPoint{Airport: "CGK", City: "Jakarta", Datetime: "2025-12-15T06:00:00+07:00", Timestamp: 1000},
		Arrival:        domain.FlightPoint{Airport: "DPS", City: "Denpasar", Datetime: "2025-12-15T08:00:00+08:00", Timestamp: 2000},
		Duration:       domain.Duration{TotalMinutes: 120, Formatted: "2h"},
		Price:          domain.Price{Amount: 500000, Currency: "IDR"},
		AvailableSeats: 10,
		CabinClass:     domain.CabinEconomy,
		Amenities:      []string{},
	}
}

func newTestCache() cache.Cache {
	return cache.NewInMemory(100)
}

func newTestLimiters(providers []provider.Provider) map[string]ratelimiter.RateLimiter {
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

func TestAggregatorReturnsAllProviders(t *testing.T) {
	providers := []provider.Provider{
		provider.NewGaruda(),
		provider.NewLionAir(),
		provider.NewBatikAir(),
		provider.NewAirAsia(),
	}
	cfg := DefaultConfig()
	agg := New(cfg, providers, newTestCache(), newTestLimiters(providers))

	// run multiple times since AirAsia has random failures
	var resp *domain.SearchResponse
	var err error
	for range 5 {
		resp, err = agg.Search(context.Background(), testReq)
		if err == nil && resp.Metadata.ProvidersSucceeded == 4 {
			break
		}
	}
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if resp == nil {
		t.Fatalf("response is nil")
	}

	if resp.Metadata.ProvidersQueried != 4 {
		t.Errorf("expected 4 providers queried, got %d", resp.Metadata.ProvidersQueried)
	}
	if resp.Metadata.TotalResults != 13 {
		t.Errorf("expected 13 flights, got %d", resp.Metadata.TotalResults)
	}
	if resp.Metadata.SearchTimeMs <= 0 {
		t.Error("search time should be positive")
	}
}

func TestAggregatorCaching(t *testing.T) {
	providers := []provider.Provider{provider.NewGaruda()}
	cfg := DefaultConfig()
	agg := New(cfg, providers, newTestCache(), newTestLimiters(providers))

	resp1, err := agg.Search(context.Background(), testReq)
	if err != nil {
		t.Fatalf("first search failed: %v", err)
	}
	if resp1.Metadata.CacheHit {
		t.Error("first search should not be a cache hit")
	}

	resp2, err := agg.Search(context.Background(), testReq)
	if err != nil {
		t.Fatalf("second search failed: %v", err)
	}
	if !resp2.Metadata.CacheHit {
		t.Error("second search should be a cache hit")
	}
}

func TestAggregatorPartialFailureNotCached(t *testing.T) {
	ctrl := gomock.NewController(t)

	successProv := mocks.NewMockProvider(ctrl)
	successProv.EXPECT().Name().Return("Good").AnyTimes()
	successProv.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]domain.Flight{validFlight()}, nil).Times(2)

	failProv := mocks.NewMockProvider(ctrl)
	failProv.EXPECT().Name().Return("Bad").AnyTimes()
	failProv.EXPECT().Search(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("down")).Times(2)

	mockRL := mocks.NewMockRateLimiter(ctrl)
	mockRL.EXPECT().Wait(gomock.Any()).Return(nil).AnyTimes()

	cfg := DefaultConfig()
	cfg.MaxRetries = 1
	rls := map[string]ratelimiter.RateLimiter{"Good": mockRL, "Bad": mockRL}
	agg := New(cfg, []provider.Provider{successProv, failProv}, newTestCache(), rls)

	resp1, err := agg.Search(context.Background(), testReq)
	if err != nil {
		t.Fatalf("first search failed: %v", err)
	}
	if resp1.Metadata.CacheHit {
		t.Error("first search should not be a cache hit")
	}

	// second search should NOT be a cache hit because the first had a failed provider
	resp2, err := agg.Search(context.Background(), testReq)
	if err != nil {
		t.Fatalf("second search failed: %v", err)
	}
	if resp2.Metadata.CacheHit {
		t.Error("partial failure results should not be cached")
	}
}

func TestAggregatorTimeout(t *testing.T) {
	providers := []provider.Provider{provider.NewBatikAir()} // 200-400ms delay > 10ms timeout
	cfg := DefaultConfig()
	cfg.ProviderTimeout = 10 * time.Millisecond
	cfg.MaxRetries = 1
	agg := New(cfg, providers, newTestCache(), newTestLimiters(providers))

	_, err := agg.Search(context.Background(), testReq)
	if err == nil {
		t.Fatal("search should return error when all providers fail")
	}
}

func TestAggregatorWithFilters(t *testing.T) {
	providers := []provider.Provider{provider.NewGaruda(), provider.NewLionAir()}
	cfg := DefaultConfig()
	agg := New(cfg, providers, newTestCache(), newTestLimiters(providers))

	maxPrice := 1000000
	req := testReq
	req.Filters = &domain.Filters{MaxPrice: &maxPrice}

	resp, err := agg.Search(context.Background(), req)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	for _, f := range resp.Flights {
		if f.Price.Amount > maxPrice {
			t.Errorf("flight %s price %d exceeds max %d", f.FlightNumber, f.Price.Amount, maxPrice)
		}
	}
}

func TestQueryWithRetry_RateLimiterError_FailOpen(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockRL := mocks.NewMockRateLimiter(ctrl)
	mockRL.EXPECT().Wait(gomock.Any()).Return(fmt.Errorf("limiter broken")).Times(1)

	mockProv := mocks.NewMockProvider(ctrl)
	mockProv.EXPECT().Name().Return("MockAir").AnyTimes()
	mockProv.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]domain.Flight{validFlight()}, nil).Times(1)

	cfg := DefaultConfig()
	cfg.MaxRetries = 1
	agg := New(cfg, []provider.Provider{mockProv}, newTestCache(), map[string]ratelimiter.RateLimiter{mockProv.Name(): mockRL})

	resp, err := agg.Search(context.Background(), testReq)
	if err != nil {
		t.Fatalf("search should not fail: %v", err)
	}
	if resp.Metadata.ProvidersSucceeded != 1 {
		t.Errorf("expected 1 succeeded provider, got %d", resp.Metadata.ProvidersSucceeded)
	}
	if resp.Metadata.TotalResults != 1 {
		t.Errorf("expected 1 flight result despite rate limiter error, got %d", resp.Metadata.TotalResults)
	}
}

func TestQueryWithRetry_ProviderFailsThenSucceeds(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockRL := mocks.NewMockRateLimiter(ctrl)
	mockRL.EXPECT().Wait(gomock.Any()).Return(nil).Times(3)

	mockProv := mocks.NewMockProvider(ctrl)
	mockProv.EXPECT().Name().Return("MockAir").AnyTimes()
	gomock.InOrder(
		mockProv.EXPECT().Search(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("temporary error")),
		mockProv.EXPECT().Search(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("temporary error")),
		mockProv.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]domain.Flight{validFlight()}, nil),
	)

	cfg := DefaultConfig()
	cfg.MaxRetries = 3
	agg := New(cfg, []provider.Provider{mockProv}, newTestCache(), map[string]ratelimiter.RateLimiter{mockProv.Name(): mockRL})

	resp, err := agg.Search(context.Background(), testReq)
	if err != nil {
		t.Fatalf("search should succeed after retries: %v", err)
	}
	if resp.Metadata.ProvidersSucceeded != 1 {
		t.Errorf("expected 1 succeeded provider, got %d", resp.Metadata.ProvidersSucceeded)
	}
	if resp.Metadata.TotalResults != 1 {
		t.Errorf("expected 1 flight, got %d", resp.Metadata.TotalResults)
	}
}

func TestQueryWithRetry_AllRetriesExhausted(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockRL := mocks.NewMockRateLimiter(ctrl)
	mockRL.EXPECT().Wait(gomock.Any()).Return(nil).Times(3)

	mockProv := mocks.NewMockProvider(ctrl)
	mockProv.EXPECT().Name().Return("MockAir").AnyTimes()
	mockProv.EXPECT().Search(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("persistent error")).Times(3)

	cfg := DefaultConfig()
	cfg.MaxRetries = 3
	agg := New(cfg, []provider.Provider{mockProv}, newTestCache(), map[string]ratelimiter.RateLimiter{mockProv.Name(): mockRL})

	_, err := agg.Search(context.Background(), testReq)
	if err == nil {
		t.Fatal("search should return error when all providers fail")
	}
}

func TestSearch_CacheHitWithFilters(t *testing.T) {
	ctrl := gomock.NewController(t)

	cheap := validFlight()
	cheap.ID = "CHEAP_Mock"
	cheap.Price.Amount = 300000

	expensive := validFlight()
	expensive.ID = "EXPENSIVE_Mock"
	expensive.Price.Amount = 900000

	mockRL := mocks.NewMockRateLimiter(ctrl)
	mockRL.EXPECT().Wait(gomock.Any()).Return(nil).AnyTimes()

	mockProv := mocks.NewMockProvider(ctrl)
	mockProv.EXPECT().Name().Return("MockAir").AnyTimes()
	mockProv.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]domain.Flight{cheap, expensive}, nil).Times(1)

	cfg := DefaultConfig()
	cfg.MaxRetries = 1
	agg := New(cfg, []provider.Provider{mockProv}, newTestCache(), map[string]ratelimiter.RateLimiter{mockProv.Name(): mockRL})

	// first search without filters — populates cache
	resp1, err := agg.Search(context.Background(), testReq)
	if err != nil {
		t.Fatalf("first search failed: %v", err)
	}
	if resp1.Metadata.CacheHit {
		t.Error("first search should not be a cache hit")
	}
	if resp1.Metadata.TotalResults != 2 {
		t.Errorf("expected 2 flights, got %d", resp1.Metadata.TotalResults)
	}

	// second search with maxPrice filter, should hit cache and apply filter
	maxPrice := 500000
	filteredReq := testReq
	filteredReq.Filters = &domain.Filters{MaxPrice: &maxPrice}
	filteredReq.SortBy = "price_asc"

	resp2, err := agg.Search(context.Background(), filteredReq)
	if err != nil {
		t.Fatalf("filtered search failed: %v", err)
	}
	if !resp2.Metadata.CacheHit {
		t.Error("second search should be a cache hit")
	}
	if resp2.Metadata.TotalResults != 1 {
		t.Errorf("expected 1 flight after filter, got %d", resp2.Metadata.TotalResults)
	}
	if resp2.Flights[0].ID != "CHEAP_Mock" {
		t.Errorf("expected the cheap flight to survive filter, got %s", resp2.Flights[0].ID)
	}
}

func TestQueryWithRetry_ContextCancelledDuringBackoff(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockRL := mocks.NewMockRateLimiter(ctrl)
	mockRL.EXPECT().Wait(gomock.Any()).Return(nil).AnyTimes()

	mockProv := mocks.NewMockProvider(ctrl)
	mockProv.EXPECT().Name().Return("MockAir").AnyTimes()
	mockProv.EXPECT().Search(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("fail")).AnyTimes()

	cfg := DefaultConfig()
	cfg.MaxRetries = 3
	agg := New(cfg, []provider.Provider{mockProv}, newTestCache(), map[string]ratelimiter.RateLimiter{mockProv.Name(): mockRL})

	// cancel the context quickly so it fires during the 500ms backoff
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := agg.Search(ctx, testReq)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("search should return error when all providers fail")
	}
	// should return well before the full backoff schedule (500ms + 1s = 1.5s)
	if elapsed > 500*time.Millisecond {
		t.Errorf("expected early return on context cancellation, but took %v", elapsed)
	}
}

func TestValidateFlights_InvalidTimestamp(t *testing.T) {
	f := validFlight()
	f.Departure.Timestamp = 5000
	f.Arrival.Timestamp = 3000 // arrival before departure

	result := validateFlights([]domain.Flight{f})
	if len(result) != 0 {
		t.Errorf("expected flight with arrival before departure to be filtered out, got %d", len(result))
	}
}

func TestValidateFlights_InvalidDuration(t *testing.T) {
	f := validFlight()
	f.Duration.TotalMinutes = 0

	result := validateFlights([]domain.Flight{f})
	if len(result) != 0 {
		t.Errorf("expected flight with zero duration to be filtered out, got %d", len(result))
	}

	f2 := validFlight()
	f2.Duration.TotalMinutes = -10

	result = validateFlights([]domain.Flight{f2})
	if len(result) != 0 {
		t.Errorf("expected flight with negative duration to be filtered out, got %d", len(result))
	}
}

func TestValidateFlights_InvalidPrice(t *testing.T) {
	f := validFlight()
	f.Price.Amount = 0

	result := validateFlights([]domain.Flight{f})
	if len(result) != 0 {
		t.Errorf("expected flight with zero price to be filtered out, got %d", len(result))
	}

	f2 := validFlight()
	f2.Price.Amount = -100

	result = validateFlights([]domain.Flight{f2})
	if len(result) != 0 {
		t.Errorf("expected flight with negative price to be filtered out, got %d", len(result))
	}
}

func TestValidateFlights_MixedValidity(t *testing.T) {
	good := validFlight()

	badTimestamp := validFlight()
	badTimestamp.Departure.Timestamp = 9000
	badTimestamp.Arrival.Timestamp = 1000

	badDuration := validFlight()
	badDuration.Duration.TotalMinutes = -5

	badPrice := validFlight()
	badPrice.Price.Amount = 0

	result := validateFlights([]domain.Flight{good, badTimestamp, badDuration, badPrice})
	if len(result) != 1 {
		t.Errorf("expected only 1 valid flight out of 4, got %d", len(result))
	}
}

func flightIDs(flights []domain.Flight) []string {
	ids := make([]string, len(flights))
	for i, f := range flights {
		ids[i] = f.ID
	}
	return ids
}

func TestFilterByRequest(t *testing.T) {
	baseFlight := func(id string) domain.Flight {
		return domain.Flight{
			ID:             id,
			Departure:      domain.FlightPoint{Airport: "CGK", Datetime: "2025-12-15T06:00:00+07:00"},
			Arrival:        domain.FlightPoint{Airport: "DPS"},
			AvailableSeats: 10,
			CabinClass:     domain.CabinEconomy,
		}
	}

	baseReq := domain.SearchRequest{
		Origin:        "CGK",
		Destination:   "DPS",
		DepartureDate: "2025-12-15",
		Passengers:    1,
		CabinClass:    domain.CabinEconomy,
	}

	tests := []struct {
		name    string
		flights []domain.Flight
		req     domain.SearchRequest
		wantIDs []string
	}{
		{
			name:    "all fields match",
			flights: []domain.Flight{baseFlight("A")},
			req:     baseReq,
			wantIDs: []string{"A"},
		},
		{
			name: "wrong departure date filtered out",
			flights: []domain.Flight{func() domain.Flight {
				f := baseFlight("A")
				f.Departure.Datetime = "2025-12-16T06:00:00+07:00"
				return f
			}()},
			req:     baseReq,
			wantIDs: []string{},
		},
		{
			name: "wrong origin filtered out",
			flights: []domain.Flight{func() domain.Flight {
				f := baseFlight("A")
				f.Departure.Airport = "SUB"
				return f
			}()},
			req:     baseReq,
			wantIDs: []string{},
		},
		{
			name: "wrong destination filtered out",
			flights: []domain.Flight{func() domain.Flight {
				f := baseFlight("A")
				f.Arrival.Airport = "SUB"
				return f
			}()},
			req:     baseReq,
			wantIDs: []string{},
		},
		{
			name: "not enough seats filtered out",
			flights: []domain.Flight{func() domain.Flight {
				f := baseFlight("A")
				f.AvailableSeats = 2
				return f
			}()},
			req: func() domain.SearchRequest {
				r := baseReq
				r.Passengers = 3
				return r
			}(),
			wantIDs: []string{},
		},
		{
			name: "exact seats match passes",
			flights: []domain.Flight{func() domain.Flight {
				f := baseFlight("A")
				f.AvailableSeats = 3
				return f
			}()},
			req: func() domain.SearchRequest {
				r := baseReq
				r.Passengers = 3
				return r
			}(),
			wantIDs: []string{"A"},
		},
		{
			name: "passengers 0 skips seat check",
			flights: []domain.Flight{func() domain.Flight {
				f := baseFlight("A")
				f.AvailableSeats = 0
				return f
			}()},
			req: func() domain.SearchRequest {
				r := baseReq
				r.Passengers = 0
				return r
			}(),
			wantIDs: []string{"A"},
		},
		{
			name: "wrong cabin class filtered out",
			flights: []domain.Flight{func() domain.Flight {
				f := baseFlight("A")
				f.CabinClass = domain.CabinBusiness
				return f
			}()},
			req:     baseReq,
			wantIDs: []string{},
		},
		{
			name: "empty cabin class in request skips class check",
			flights: []domain.Flight{func() domain.Flight {
				f := baseFlight("A")
				f.CabinClass = domain.CabinBusiness
				return f
			}()},
			req: func() domain.SearchRequest {
				r := baseReq
				r.CabinClass = ""
				return r
			}(),
			wantIDs: []string{"A"},
		},
		{
			name: "mixed flights",
			flights: []domain.Flight{
				baseFlight("A"),
				func() domain.Flight {
					f := baseFlight("B")
					f.Departure.Datetime = "2025-12-16T06:00:00+07:00"
					return f
				}(),
				func() domain.Flight {
					f := baseFlight("C")
					f.Departure.Airport = "DPS"
					f.Arrival.Airport = "CGK"
					return f
				}(),
				func() domain.Flight {
					f := baseFlight("D")
					f.CabinClass = domain.CabinBusiness
					return f
				}(),
				baseFlight("E"),
			},
			req:     baseReq,
			wantIDs: []string{"A", "E"},
		},
		{
			name:    "empty input returns empty",
			flights: []domain.Flight{},
			req:     baseReq,
			wantIDs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterByRequest(tt.flights, tt.req)
			gotIDs := flightIDs(result)
			if !reflect.DeepEqual(gotIDs, tt.wantIDs) {
				t.Errorf("got IDs %v, want %v", gotIDs, tt.wantIDs)
			}
		})
	}
}
