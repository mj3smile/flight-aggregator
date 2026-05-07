package provider

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mj3smile/flight-aggregator/internal/domain"
)

//go:embed mockdata/lion_air_search_response.json
var lionAirData []byte

type lionResponse struct {
	Success bool     `json:"success"`
	Data    lionData `json:"data"`
}

type lionData struct {
	Flights []lionFlight `json:"available_flights"`
}

type lionFlight struct {
	ID         string        `json:"id"`
	Carrier    lionCarrier   `json:"carrier"`
	Route      lionRoute     `json:"route"`
	Schedule   lionSchedule  `json:"schedule"`
	FlightTime int           `json:"flight_time"`
	IsDirect   bool          `json:"is_direct"`
	StopCount  int           `json:"stop_count"`
	Layovers   []lionLayover `json:"layovers"`
	Pricing    lionPricing   `json:"pricing"`
	SeatsLeft  int           `json:"seats_left"`
	PlaneType  string        `json:"plane_type"`
	Services   lionServices  `json:"services"`
}

type lionCarrier struct {
	Name string `json:"name"`
	IATA string `json:"iata"`
}

type lionRoute struct {
	From lionAirport `json:"from"`
	To   lionAirport `json:"to"`
}

type lionAirport struct {
	Code string `json:"code"`
	Name string `json:"name"`
	City string `json:"city"`
}

type lionSchedule struct {
	Departure   string `json:"departure"`
	DepTimezone string `json:"departure_timezone"`
	Arrival     string `json:"arrival"`
	ArrTimezone string `json:"arrival_timezone"`
}

type lionLayover struct {
	Airport  string `json:"airport"`
	Duration int    `json:"duration_minutes"`
}

type lionPricing struct {
	Total    int    `json:"total"`
	Currency string `json:"currency"`
	FareType string `json:"fare_type"`
}

type lionServices struct {
	WiFi    bool        `json:"wifi_available"`
	Meals   bool        `json:"meals_included"`
	Baggage lionBaggage `json:"baggage_allowance"`
}

type lionBaggage struct {
	Cabin string `json:"cabin"`
	Hold  string `json:"hold"`
}

type LionAirProvider struct{}

func NewLionAir() *LionAirProvider { return &LionAirProvider{} }

func (l *LionAirProvider) Name() string { return "Lion Air" }

func (l *LionAirProvider) RateLimit() *RateLimitConfig {
	return &RateLimitConfig{Burst: 8, Interval: time.Second}
}

func (l *LionAirProvider) Search(ctx context.Context, req domain.SearchRequest) ([]domain.Flight, error) {
	if err := simulateDelay(ctx, 100, 200); err != nil {
		return nil, err
	}

	var resp lionResponse
	if err := json.Unmarshal(lionAirData, &resp); err != nil {
		return nil, fmt.Errorf("lionair: unmarshal: %w", err)
	}
	if !resp.Success {
		return nil, fmt.Errorf("lionair: api returned failure")
	}

	flights := make([]domain.Flight, 0, len(resp.Data.Flights))
	for _, lf := range resp.Data.Flights {
		f, err := l.normalize(lf)
		if err != nil {
			continue
		}
		flights = append(flights, f)
	}
	return flights, nil
}

func (l *LionAirProvider) normalize(lf lionFlight) (domain.Flight, error) {
	depTime, err := parseTimeWithLocation(lf.Schedule.Departure, lf.Schedule.DepTimezone)
	if err != nil {
		return domain.Flight{}, fmt.Errorf("parse departure: %w", err)
	}

	arrTime, err := parseTimeWithLocation(lf.Schedule.Arrival, lf.Schedule.ArrTimezone)
	if err != nil {
		return domain.Flight{}, fmt.Errorf("parse arrival: %w", err)
	}

	totalMin := domain.CalcDurationMinutes(depTime, arrTime)

	stops := 0
	if !lf.IsDirect {
		stops = lf.StopCount
	}

	var amenities []string
	if lf.Services.WiFi {
		amenities = append(amenities, "wifi")
	}
	if lf.Services.Meals {
		amenities = append(amenities, "meal")
	}
	if amenities == nil {
		amenities = []string{}
	}

	fareClass := domain.CabinEconomy
	if lf.Pricing.FareType != "" {
		switch lf.Pricing.FareType {
		case "ECONOMY":
			fareClass = domain.CabinEconomy
		case "BUSINESS":
			fareClass = domain.CabinBusiness
		case "FIRST":
			fareClass = domain.CabinFirst
		default:
			fareClass = lf.Pricing.FareType
		}
	}

	return domain.Flight{
		ID:           fmt.Sprintf("%s_%s", lf.ID, lf.Carrier.Name),
		Provider:     lf.Carrier.Name,
		Airline:      domain.Airline{Name: lf.Carrier.Name, Code: lf.Carrier.IATA},
		FlightNumber: lf.ID,
		Departure:    makeFlightPoint(lf.Route.From.Code, depTime),
		Arrival:      makeFlightPoint(lf.Route.To.Code, arrTime),
		Duration: domain.Duration{
			TotalMinutes: totalMin,
			Formatted:    domain.FormatDuration(totalMin),
		},
		Stops: stops,
		Price: domain.Price{
			Amount:   lf.Pricing.Total,
			Currency: lf.Pricing.Currency,
			Display:  domain.FormatIDR(lf.Pricing.Total),
		},
		AvailableSeats: lf.SeatsLeft,
		CabinClass:     fareClass,
		Aircraft:       ptr(lf.PlaneType),
		Amenities:      amenities,
		Baggage: domain.Baggage{
			CarryOn: lf.Services.Baggage.Cabin,
			Checked: lf.Services.Baggage.Hold,
		},
	}, nil
}
