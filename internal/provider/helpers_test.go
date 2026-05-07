package provider

import (
	"time"

	"github.com/mj3smile/flight-aggregator/internal/domain"
)

func mustParseRFC3339(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func mustParseCompactOffset(s string) time.Time {
	t, err := time.Parse("2006-01-02T15:04:05-0700", s)
	if err != nil {
		panic(err)
	}
	return t
}

func mustParseWithLocation(s, tz string) time.Time {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		panic(err)
	}
	t, err := time.ParseInLocation("2006-01-02T15:04:05", s, loc)
	if err != nil {
		panic(err)
	}
	return t
}

func flightPoint(airport string, t time.Time) domain.FlightPoint {
	return domain.FlightPoint{
		Airport:   airport,
		City:      domain.CityForAirport(airport),
		Datetime:  t.Format(time.RFC3339),
		Timestamp: t.Unix(),
	}
}
