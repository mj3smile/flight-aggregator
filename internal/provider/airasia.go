package provider

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/mj3smile/flight-aggregator/internal/domain"
)

//go:embed mockdata/airasia_search_response.json
var airAsiaData []byte

type airAsiaResponse struct {
	Status  string          `json:"status"`
	Flights []airAsiaFlight `json:"flights"`
}

type airAsiaFlight struct {
	FlightCode   string        `json:"flight_code"`
	Airline      string        `json:"airline"`
	FromAirport  string        `json:"from_airport"`
	ToAirport    string        `json:"to_airport"`
	DepartTime   string        `json:"depart_time"`
	ArriveTime   string        `json:"arrive_time"`
	DurationHrs  float64       `json:"duration_hours"`
	DirectFlight bool          `json:"direct_flight"`
	PriceIDR     int           `json:"price_idr"`
	Seats        int           `json:"seats"`
	CabinClass   string        `json:"cabin_class"`
	BaggageNote  string        `json:"baggage_note"`
	Stops        []airAsiaStop `json:"stops"`
}

type airAsiaStop struct {
	Airport  string `json:"airport"`
	WaitTime int    `json:"wait_time_minutes"`
}

type AirAsiaProvider struct{}

func NewAirAsia() *AirAsiaProvider { return &AirAsiaProvider{} }

func (a *AirAsiaProvider) Name() string { return "AirAsia" }

func (a *AirAsiaProvider) RateLimit() *RateLimitConfig {
	return &RateLimitConfig{Burst: 15, Interval: time.Second}
}

func (a *AirAsiaProvider) Search(ctx context.Context, req domain.SearchRequest) ([]domain.Flight, error) {
	if err := simulateDelay(ctx, 50, 150); err != nil {
		return nil, err
	}

	// 90% success rate
	if rand.Float64() > 0.9 {
		return nil, fmt.Errorf("airasia: service temporarily unavailable")
	}

	var resp airAsiaResponse
	if err := json.Unmarshal(airAsiaData, &resp); err != nil {
		return nil, fmt.Errorf("airasia: unmarshal: %w", err)
	}
	if resp.Status != "ok" {
		return nil, fmt.Errorf("airasia: api returned status %s", resp.Status)
	}

	flights := make([]domain.Flight, 0, len(resp.Flights))
	for _, af := range resp.Flights {
		f, err := a.normalize(af)
		if err != nil {
			continue
		}
		flights = append(flights, f)
	}
	return flights, nil
}

func (a *AirAsiaProvider) normalize(af airAsiaFlight) (domain.Flight, error) {
	depTime, err := parseTimeRFC3339(af.DepartTime)
	if err != nil {
		return domain.Flight{}, fmt.Errorf("parse departure: %w", err)
	}

	arrTime, err := parseTimeRFC3339(af.ArriveTime)
	if err != nil {
		return domain.Flight{}, fmt.Errorf("parse arrival: %w", err)
	}

	totalMin := domain.CalcDurationMinutes(depTime, arrTime)

	stops := 0
	if !af.DirectFlight {
		stops = len(af.Stops)
		if stops == 0 {
			stops = 1
		}
	}

	carryOn, checked := a.parseBaggage(af.BaggageNote)

	return domain.Flight{
		ID:           fmt.Sprintf("%s_%s", af.FlightCode, af.Airline),
		Provider:     af.Airline,
		Airline:      domain.Airline{Name: af.Airline, Code: af.FlightCode[:2]},
		FlightNumber: af.FlightCode,
		Departure:    makeFlightPoint(af.FromAirport, depTime),
		Arrival:      makeFlightPoint(af.ToAirport, arrTime),
		Duration: domain.Duration{
			TotalMinutes: totalMin,
			Formatted:    domain.FormatDuration(totalMin),
		},
		Stops: stops,
		Price: domain.Price{
			Amount:   af.PriceIDR,
			Currency: "IDR",
			Display:  domain.FormatIDR(af.PriceIDR),
		},
		AvailableSeats: af.Seats,
		CabinClass:     af.CabinClass,
		Aircraft:       nil,
		Amenities:      []string{},
		Baggage: domain.Baggage{
			CarryOn: carryOn,
			Checked: checked,
		},
	}, nil
}

func (a *AirAsiaProvider) parseBaggage(note string) (carryOn, checked string) {
	parts := strings.SplitN(note, ",", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return note, ""
}
