package provider

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mj3smile/flight-aggregator/internal/domain"
)

//go:embed mockdata/garuda_indonesia_search_response.json
var garudaData []byte

type garudaResponse struct {
	Status  string         `json:"status"`
	Flights []garudaFlight `json:"flights"`
}

type garudaFlight struct {
	FlightID    string          `json:"flight_id"`
	Airline     string          `json:"airline"`
	AirlineCode string          `json:"airline_code"`
	Departure   garudaEndpoint  `json:"departure"`
	Arrival     garudaEndpoint  `json:"arrival"`
	DurationMin int             `json:"duration_minutes"`
	Stops       int             `json:"stops"`
	Aircraft    string          `json:"aircraft"`
	Price       garudaPrice     `json:"price"`
	Seats       int             `json:"available_seats"`
	FareClass   string          `json:"fare_class"`
	Baggage     garudaBaggage   `json:"baggage"`
	Amenities   []string        `json:"amenities"`
	Segments    []garudaSegment `json:"segments"`
}

type garudaEndpoint struct {
	Airport  string `json:"airport"`
	City     string `json:"city"`
	Time     string `json:"time"`
	Terminal string `json:"terminal"`
}

type garudaPrice struct {
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
}

type garudaBaggage struct {
	CarryOn int `json:"carry_on"`
	Checked int `json:"checked"`
}

type garudaSegment struct {
	FlightNumber string         `json:"flight_number"`
	Departure    garudaSegPoint `json:"departure"`
	Arrival      garudaSegPoint `json:"arrival"`
	DurationMin  int            `json:"duration_minutes"`
	LayoverMin   int            `json:"layover_minutes"`
}

type garudaSegPoint struct {
	Airport string `json:"airport"`
	Time    string `json:"time"`
}

type GarudaProvider struct{}

func NewGaruda() *GarudaProvider { return &GarudaProvider{} }

func (g *GarudaProvider) Name() string { return "Garuda Indonesia" }

func (g *GarudaProvider) RateLimit() *RateLimitConfig {
	return &RateLimitConfig{Burst: 10, Interval: time.Second}
}

func (g *GarudaProvider) Search(ctx context.Context, req domain.SearchRequest) ([]domain.Flight, error) {
	if err := simulateDelay(ctx, 50, 100); err != nil {
		return nil, err
	}

	var resp garudaResponse
	if err := json.Unmarshal(garudaData, &resp); err != nil {
		return nil, fmt.Errorf("garuda: unmarshal: %w", err)
	}
	if resp.Status != "success" {
		return nil, fmt.Errorf("garuda: api returned status %s", resp.Status)
	}

	flights := make([]domain.Flight, 0, len(resp.Flights))
	for _, gf := range resp.Flights {
		f, err := g.normalize(gf)
		if err != nil {
			continue
		}
		flights = append(flights, f)
	}
	return flights, nil
}

func (g *GarudaProvider) normalize(gf garudaFlight) (domain.Flight, error) {
	depTime, err := parseTimeRFC3339(gf.Departure.Time)
	if err != nil {
		return domain.Flight{}, fmt.Errorf("parse departure: %w", err)
	}

	arrTime, err := parseTimeRFC3339(gf.Arrival.Time)
	if err != nil {
		return domain.Flight{}, fmt.Errorf("parse arrival: %w", err)
	}

	arrAirport := gf.Arrival.Airport
	stops := gf.Stops

	// handle data inconsistency: if segments exist, derive actual arrival and stops from them
	if len(gf.Segments) > 1 {
		lastSeg := gf.Segments[len(gf.Segments)-1]
		parsed, err := parseTimeRFC3339(lastSeg.Arrival.Time)
		if err == nil {
			arrTime = parsed
			arrAirport = lastSeg.Arrival.Airport
		}
		stops = len(gf.Segments) - 1
	}

	totalMin := domain.CalcDurationMinutes(depTime, arrTime)

	bagCarryOn := fmt.Sprintf("%d piece", gf.Baggage.CarryOn)
	if gf.Baggage.CarryOn > 1 {
		bagCarryOn += "s"
	}
	bagChecked := fmt.Sprintf("%d piece", gf.Baggage.Checked)
	if gf.Baggage.Checked > 1 {
		bagChecked += "s"
	}

	amenities := gf.Amenities
	if amenities == nil {
		amenities = []string{}
	}

	return domain.Flight{
		ID:           fmt.Sprintf("%s_%s", gf.FlightID, gf.Airline),
		Provider:     gf.Airline,
		Airline:      domain.Airline{Name: gf.Airline, Code: gf.AirlineCode},
		FlightNumber: gf.FlightID,
		Departure:    makeFlightPoint(gf.Departure.Airport, depTime),
		Arrival:      makeFlightPoint(arrAirport, arrTime),
		Duration: domain.Duration{
			TotalMinutes: totalMin,
			Formatted:    domain.FormatDuration(totalMin),
		},
		Stops: stops,
		Price: domain.Price{
			Amount:   gf.Price.Amount,
			Currency: gf.Price.Currency,
			Display:  domain.FormatIDR(gf.Price.Amount),
		},
		AvailableSeats: gf.Seats,
		CabinClass:     gf.FareClass,
		Aircraft:       ptr(gf.Aircraft),
		Amenities:      amenities,
		Baggage: domain.Baggage{
			CarryOn: bagCarryOn,
			Checked: bagChecked,
		},
	}, nil
}
