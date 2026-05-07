package domain

import (
	"fmt"
	"time"
)

const (
	CabinEconomy = "economy"
	CabinBusiness = "business"
	CabinFirst    = "first"
)

type SearchRequest struct {
	Origin        string   `json:"origin"`
	Destination   string   `json:"destination"`
	DepartureDate string   `json:"departureDate"`
	ReturnDate    *string  `json:"returnDate"`
	Passengers    int      `json:"passengers"`
	CabinClass    string   `json:"cabinClass"`
	Filters       *Filters `json:"filters,omitempty"`
	SortBy        string   `json:"sortBy,omitempty"`
}

type Filters struct {
	MinPrice        *int     `json:"minPrice,omitempty"`
	MaxPrice        *int     `json:"maxPrice,omitempty"`
	MaxStops        *int     `json:"maxStops,omitempty"`
	Airlines        []string `json:"airlines,omitempty"`
	DepartureAfter  *string  `json:"departureAfter,omitempty"`
	DepartureBefore *string  `json:"departureBefore,omitempty"`
	ArrivalAfter    *string  `json:"arrivalAfter,omitempty"`
	ArrivalBefore   *string  `json:"arrivalBefore,omitempty"`
	MaxDuration     *int     `json:"maxDuration,omitempty"`
}

type SearchResponse struct {
	SearchCriteria SearchCriteria `json:"search_criteria"`
	Metadata       Metadata       `json:"metadata"`
	Flights        []Flight       `json:"flights"`
}

type SearchCriteria struct {
	Origin        string `json:"origin"`
	Destination   string `json:"destination"`
	DepartureDate string `json:"departure_date"`
	Passengers    int    `json:"passengers"`
	CabinClass    string `json:"cabin_class"`
}

type Metadata struct {
	TotalResults       int   `json:"total_results"`
	ProvidersQueried   int   `json:"providers_queried"`
	ProvidersSucceeded int   `json:"providers_succeeded"`
	ProvidersFailed    int   `json:"providers_failed"`
	SearchTimeMs       int64 `json:"search_time_ms"`
	CacheHit           bool  `json:"cache_hit"`
}

type Flight struct {
	ID             string      `json:"id"`
	Provider       string      `json:"provider"`
	Airline        Airline     `json:"airline"`
	FlightNumber   string      `json:"flight_number"`
	Departure      FlightPoint `json:"departure"`
	Arrival        FlightPoint `json:"arrival"`
	Duration       Duration    `json:"duration"`
	Stops          int         `json:"stops"`
	Price          Price       `json:"price"`
	AvailableSeats int         `json:"available_seats"`
	CabinClass     string      `json:"cabin_class"`
	Aircraft       *string     `json:"aircraft"`
	Amenities      []string    `json:"amenities"`
	Baggage        Baggage     `json:"baggage"`
	Score          *float64    `json:"score,omitempty"`
}

type Airline struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

type FlightPoint struct {
	Airport   string `json:"airport"`
	City      string `json:"city"`
	Datetime  string `json:"datetime"`
	Timestamp int64  `json:"timestamp"`
}

type Duration struct {
	TotalMinutes int    `json:"total_minutes"`
	Formatted    string `json:"formatted"`
}

type Price struct {
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
	Display  string `json:"display"`
}

type Baggage struct {
	CarryOn string `json:"carry_on"`
	Checked string `json:"checked"`
}

func FormatDuration(minutes int) string {
	h := minutes / 60
	m := minutes % 60
	if h == 0 {
		return fmt.Sprintf("%dm", m)
	}
	if m == 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dh %dm", h, m)
}

func FormatIDR(amount int) string {
	s := fmt.Sprintf("%d", amount)
	n := len(s)
	if n <= 3 {
		return "IDR " + s
	}
	var result []byte
	for i := 0; i < n; i++ {
		if i > 0 && (n-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, s[i])
	}
	return "IDR " + string(result)
}

func CalcDurationMinutes(dep, arr time.Time) int {
	return int(arr.Sub(dep).Minutes())
}

func CacheKey(req SearchRequest) string {
	return fmt.Sprintf("%s:%s:%s:%d:%s",
		req.Origin, req.Destination, req.DepartureDate,
		req.Passengers, req.CabinClass)
}
