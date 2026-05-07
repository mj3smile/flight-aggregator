package provider

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mj3smile/flight-aggregator/internal/domain"
)

//go:embed mockdata/batik_air_search_response.json
var batikAirData []byte

type batikResponse struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Results []batikFlight `json:"results"`
}

type batikFlight struct {
	FlightNumber string            `json:"flightNumber"`
	AirlineName  string            `json:"airlineName"`
	AirlineIATA  string            `json:"airlineIATA"`
	Origin       string            `json:"origin"`
	Destination  string            `json:"destination"`
	DepDateTime  string            `json:"departureDateTime"`
	ArrDateTime  string            `json:"arrivalDateTime"`
	TravelTime   string            `json:"travelTime"`
	Stops        int               `json:"numberOfStops"`
	Connections  []batikConnection `json:"connections"`
	Fare         batikFare         `json:"fare"`
	Seats        int               `json:"seatsAvailable"`
	Aircraft     string            `json:"aircraftModel"`
	BaggageInfo  string            `json:"baggageInfo"`
	Services     []string          `json:"onboardServices"`
}

type batikConnection struct {
	StopAirport  string `json:"stopAirport"`
	StopDuration string `json:"stopDuration"`
}

type batikFare struct {
	BasePrice int    `json:"basePrice"`
	Taxes     int    `json:"taxes"`
	Total     int    `json:"totalPrice"`
	Currency  string `json:"currencyCode"`
	Class     string `json:"class"`
}

type BatikAirProvider struct{}

func NewBatikAir() *BatikAirProvider { return &BatikAirProvider{} }

func (b *BatikAirProvider) Name() string { return "Batik Air" }

func (b *BatikAirProvider) RateLimit() *RateLimitConfig {
	return &RateLimitConfig{Burst: 5, Interval: time.Second}
}

func (b *BatikAirProvider) Search(ctx context.Context, req domain.SearchRequest) ([]domain.Flight, error) {
	if err := simulateDelay(ctx, 200, 400); err != nil {
		return nil, err
	}

	var resp batikResponse
	if err := json.Unmarshal(batikAirData, &resp); err != nil {
		return nil, fmt.Errorf("batikair: unmarshal: %w", err)
	}
	if resp.Code != 200 {
		return nil, fmt.Errorf("batikair: api returned code %d", resp.Code)
	}

	flights := make([]domain.Flight, 0, len(resp.Results))
	for _, bf := range resp.Results {
		f, err := b.normalize(bf)
		if err != nil {
			continue
		}
		flights = append(flights, f)
	}
	return flights, nil
}

func (b *BatikAirProvider) normalize(bf batikFlight) (domain.Flight, error) {
	depTime, err := parseTimeCompactOffset(bf.DepDateTime)
	if err != nil {
		return domain.Flight{}, fmt.Errorf("parse departure: %w", err)
	}

	arrTime, err := parseTimeCompactOffset(bf.ArrDateTime)
	if err != nil {
		return domain.Flight{}, fmt.Errorf("parse arrival: %w", err)
	}

	totalMin := domain.CalcDurationMinutes(depTime, arrTime)

	fareClass := domain.CabinEconomy
	switch bf.Fare.Class {
	case "Y":
		fareClass = domain.CabinEconomy
	case "C":
		fareClass = domain.CabinBusiness
	case "F":
		fareClass = domain.CabinFirst
	}

	carryOn, checked := b.parseBaggageInfo(bf.BaggageInfo)

	services := bf.Services
	if services == nil {
		services = []string{}
	}
	amenities := make([]string, 0, len(services))
	for _, s := range services {
		amenities = append(amenities, strings.ToLower(s))
	}

	return domain.Flight{
		ID:           fmt.Sprintf("%s_%s", bf.FlightNumber, bf.AirlineName),
		Provider:     bf.AirlineName,
		Airline:      domain.Airline{Name: bf.AirlineName, Code: bf.AirlineIATA},
		FlightNumber: bf.FlightNumber,
		Departure:    makeFlightPoint(bf.Origin, depTime),
		Arrival:      makeFlightPoint(bf.Destination, arrTime),
		Duration: domain.Duration{
			TotalMinutes: totalMin,
			Formatted:    domain.FormatDuration(totalMin),
		},
		Stops: bf.Stops,
		Price: domain.Price{
			Amount:   bf.Fare.Total,
			Currency: bf.Fare.Currency,
			Display:  domain.FormatIDR(bf.Fare.Total),
		},
		AvailableSeats: bf.Seats,
		CabinClass:     fareClass,
		Aircraft:       ptr(bf.Aircraft),
		Amenities:      amenities,
		Baggage: domain.Baggage{
			CarryOn: carryOn,
			Checked: checked,
		},
	}, nil
}

func (b *BatikAirProvider) parseBaggageInfo(info string) (carryOn, checked string) {
	parts := strings.SplitN(info, ",", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return info, ""
}
