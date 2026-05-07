package provider

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/mj3smile/flight-aggregator/internal/domain"
)

type RateLimitConfig struct {
	Burst    int
	Interval time.Duration
}

type Provider interface {
	Name() string
	Search(ctx context.Context, req domain.SearchRequest) ([]domain.Flight, error)
	RateLimit() *RateLimitConfig
}

func simulateDelay(ctx context.Context, minMs, maxMs int) error {
	delay := time.Duration(minMs+rand.IntN(maxMs-minMs+1)) * time.Millisecond
	select {
	case <-time.After(delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func parseTimeRFC3339(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

func parseTimeCompactOffset(s string) (time.Time, error) {
	return time.Parse("2006-01-02T15:04:05-0700", s)
}

func parseTimeWithLocation(s, tz string) (time.Time, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.Time{}, err
	}
	t, err := time.ParseInLocation("2006-01-02T15:04:05", s, loc)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

func makeFlightPoint(airport string, t time.Time) domain.FlightPoint {
	return domain.FlightPoint{
		Airport:   airport,
		City:      domain.CityForAirport(airport),
		Datetime:  t.Format(time.RFC3339),
		Timestamp: t.Unix(),
	}
}

func ptr(s string) *string { return &s }
