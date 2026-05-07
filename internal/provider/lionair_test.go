package provider

import (
	"reflect"
	"testing"

	"github.com/mj3smile/flight-aggregator/internal/domain"
)

func TestLionAirNormalize(t *testing.T) {
	p := &LionAirProvider{}

	tests := []struct {
		name    string
		input   lionFlight
		want    domain.Flight
		wantErr bool
	}{
		{
			name: "direct flight with IANA timezone",
			input: lionFlight{
				ID:        "JT740",
				Carrier:   lionCarrier{Name: "Lion Air", IATA: "JT"},
				Route:     lionRoute{From: lionAirport{Code: "CGK"}, To: lionAirport{Code: "DPS"}},
				Schedule:  lionSchedule{Departure: "2025-12-15T05:30:00", DepTimezone: "Asia/Jakarta", Arrival: "2025-12-15T08:15:00", ArrTimezone: "Asia/Makassar"},
				IsDirect:  true,
				Pricing:   lionPricing{Total: 950000, Currency: "IDR", FareType: "ECONOMY"},
				SeatsLeft: 45,
				PlaneType: "Boeing 737-900ER",
				Services: lionServices{
					WiFi:    false,
					Meals:   false,
					Baggage: lionBaggage{Cabin: "7 kg", Hold: "20 kg"},
				},
			},
			want: domain.Flight{
				ID:             "JT740_Lion Air",
				Provider:       "Lion Air",
				Airline:        domain.Airline{Name: "Lion Air", Code: "JT"},
				FlightNumber:   "JT740",
				Departure:      flightPoint("CGK", mustParseWithLocation("2025-12-15T05:30:00", "Asia/Jakarta")),
				Arrival:        flightPoint("DPS", mustParseWithLocation("2025-12-15T08:15:00", "Asia/Makassar")),
				Duration:       domain.Duration{TotalMinutes: 105, Formatted: "1h 45m"},
				Stops:          0,
				Price:          domain.Price{Amount: 950000, Currency: "IDR", Display: "IDR 950,000"},
				AvailableSeats: 45,
				CabinClass:     domain.CabinEconomy,
				Aircraft:       ptr("Boeing 737-900ER"),
				Amenities:      []string{},
				Baggage:        domain.Baggage{CarryOn: "7 kg", Checked: "20 kg"},
			},
		},
		{
			name: "non-direct with stop count",
			input: lionFlight{
				ID:        "JT650",
				Carrier:   lionCarrier{Name: "Lion Air", IATA: "JT"},
				Route:     lionRoute{From: lionAirport{Code: "CGK"}, To: lionAirport{Code: "DPS"}},
				Schedule:  lionSchedule{Departure: "2025-12-15T16:20:00", DepTimezone: "Asia/Jakarta", Arrival: "2025-12-15T21:10:00", ArrTimezone: "Asia/Makassar"},
				IsDirect:  false,
				StopCount: 1,
				Pricing:   lionPricing{Total: 780000, Currency: "IDR", FareType: "ECONOMY"},
				SeatsLeft: 52,
				PlaneType: "Boeing 737-800",
				Services:  lionServices{Baggage: lionBaggage{Cabin: "7 kg", Hold: "20 kg"}},
			},
			want: domain.Flight{
				ID:             "JT650_Lion Air",
				Provider:       "Lion Air",
				Airline:        domain.Airline{Name: "Lion Air", Code: "JT"},
				FlightNumber:   "JT650",
				Departure:      flightPoint("CGK", mustParseWithLocation("2025-12-15T16:20:00", "Asia/Jakarta")),
				Arrival:        flightPoint("DPS", mustParseWithLocation("2025-12-15T21:10:00", "Asia/Makassar")),
				Duration:       domain.Duration{TotalMinutes: 230, Formatted: "3h 50m"},
				Stops:          1,
				Price:          domain.Price{Amount: 780000, Currency: "IDR", Display: "IDR 780,000"},
				AvailableSeats: 52,
				CabinClass:     domain.CabinEconomy,
				Aircraft:       ptr("Boeing 737-800"),
				Amenities:      []string{},
				Baggage:        domain.Baggage{CarryOn: "7 kg", Checked: "20 kg"},
			},
		},
		{
			name: "business class fare type",
			input: lionFlight{
				ID:        "JT100",
				Carrier:   lionCarrier{Name: "Lion Air", IATA: "JT"},
				Route:     lionRoute{From: lionAirport{Code: "CGK"}, To: lionAirport{Code: "DPS"}},
				Schedule:  lionSchedule{Departure: "2025-12-15T06:00:00", DepTimezone: "Asia/Jakarta", Arrival: "2025-12-15T08:00:00", ArrTimezone: "Asia/Makassar"},
				IsDirect:  true,
				Pricing:   lionPricing{Total: 2500000, Currency: "IDR", FareType: "BUSINESS"},
				SeatsLeft: 8,
				PlaneType: "A330",
				Services:  lionServices{WiFi: true, Meals: true, Baggage: lionBaggage{Cabin: "10 kg", Hold: "30 kg"}},
			},
			want: domain.Flight{
				ID:             "JT100_Lion Air",
				Provider:       "Lion Air",
				Airline:        domain.Airline{Name: "Lion Air", Code: "JT"},
				FlightNumber:   "JT100",
				Departure:      flightPoint("CGK", mustParseWithLocation("2025-12-15T06:00:00", "Asia/Jakarta")),
				Arrival:        flightPoint("DPS", mustParseWithLocation("2025-12-15T08:00:00", "Asia/Makassar")),
				Duration:       domain.Duration{TotalMinutes: 60, Formatted: "1h"},
				Stops:          0,
				Price:          domain.Price{Amount: 2500000, Currency: "IDR", Display: "IDR 2,500,000"},
				AvailableSeats: 8,
				CabinClass:     domain.CabinBusiness,
				Aircraft:       ptr("A330"),
				Amenities:      []string{"wifi", "meal"},
				Baggage:        domain.Baggage{CarryOn: "10 kg", Checked: "30 kg"},
			},
		},
		{
			name: "first class fare type",
			input: lionFlight{
				ID:        "JT200",
				Carrier:   lionCarrier{Name: "Lion Air", IATA: "JT"},
				Route:     lionRoute{From: lionAirport{Code: "CGK"}, To: lionAirport{Code: "DPS"}},
				Schedule:  lionSchedule{Departure: "2025-12-15T06:00:00", DepTimezone: "Asia/Jakarta", Arrival: "2025-12-15T08:00:00", ArrTimezone: "Asia/Makassar"},
				IsDirect:  true,
				Pricing:   lionPricing{Total: 5000000, Currency: "IDR", FareType: "FIRST"},
				SeatsLeft: 2,
				PlaneType: "A380",
				Services:  lionServices{Baggage: lionBaggage{Cabin: "15 kg", Hold: "40 kg"}},
			},
			want: domain.Flight{
				ID:             "JT200_Lion Air",
				Provider:       "Lion Air",
				Airline:        domain.Airline{Name: "Lion Air", Code: "JT"},
				FlightNumber:   "JT200",
				Departure:      flightPoint("CGK", mustParseWithLocation("2025-12-15T06:00:00", "Asia/Jakarta")),
				Arrival:        flightPoint("DPS", mustParseWithLocation("2025-12-15T08:00:00", "Asia/Makassar")),
				Duration:       domain.Duration{TotalMinutes: 60, Formatted: "1h"},
				Stops:          0,
				Price:          domain.Price{Amount: 5000000, Currency: "IDR", Display: "IDR 5,000,000"},
				AvailableSeats: 2,
				CabinClass:     domain.CabinFirst,
				Aircraft:       ptr("A380"),
				Amenities:      []string{},
				Baggage:        domain.Baggage{CarryOn: "15 kg", Checked: "40 kg"},
			},
		},
		{
			name: "unknown fare type passed through",
			input: lionFlight{
				ID:        "JT300",
				Carrier:   lionCarrier{Name: "Lion Air", IATA: "JT"},
				Route:     lionRoute{From: lionAirport{Code: "CGK"}, To: lionAirport{Code: "DPS"}},
				Schedule:  lionSchedule{Departure: "2025-12-15T06:00:00", DepTimezone: "Asia/Jakarta", Arrival: "2025-12-15T08:00:00", ArrTimezone: "Asia/Makassar"},
				IsDirect:  true,
				Pricing:   lionPricing{Total: 600000, Currency: "IDR", FareType: "PROMO"},
				SeatsLeft: 100,
				PlaneType: "737",
				Services:  lionServices{Baggage: lionBaggage{Cabin: "7 kg", Hold: "15 kg"}},
			},
			want: domain.Flight{
				ID:             "JT300_Lion Air",
				Provider:       "Lion Air",
				Airline:        domain.Airline{Name: "Lion Air", Code: "JT"},
				FlightNumber:   "JT300",
				Departure:      flightPoint("CGK", mustParseWithLocation("2025-12-15T06:00:00", "Asia/Jakarta")),
				Arrival:        flightPoint("DPS", mustParseWithLocation("2025-12-15T08:00:00", "Asia/Makassar")),
				Duration:       domain.Duration{TotalMinutes: 60, Formatted: "1h"},
				Stops:          0,
				Price:          domain.Price{Amount: 600000, Currency: "IDR", Display: "IDR 600,000"},
				AvailableSeats: 100,
				CabinClass:     "PROMO",
				Aircraft:       ptr("737"),
				Amenities:      []string{},
				Baggage:        domain.Baggage{CarryOn: "7 kg", Checked: "15 kg"},
			},
		},
		{
			name: "wifi and meals amenities",
			input: lionFlight{
				ID:        "JT400",
				Carrier:   lionCarrier{Name: "Lion Air", IATA: "JT"},
				Route:     lionRoute{From: lionAirport{Code: "CGK"}, To: lionAirport{Code: "DPS"}},
				Schedule:  lionSchedule{Departure: "2025-12-15T06:00:00", DepTimezone: "Asia/Jakarta", Arrival: "2025-12-15T08:00:00", ArrTimezone: "Asia/Makassar"},
				IsDirect:  true,
				Pricing:   lionPricing{Total: 1000000, Currency: "IDR", FareType: "ECONOMY"},
				SeatsLeft: 20,
				PlaneType: "A320",
				Services:  lionServices{WiFi: true, Meals: true, Baggage: lionBaggage{Cabin: "7 kg", Hold: "20 kg"}},
			},
			want: domain.Flight{
				ID:             "JT400_Lion Air",
				Provider:       "Lion Air",
				Airline:        domain.Airline{Name: "Lion Air", Code: "JT"},
				FlightNumber:   "JT400",
				Departure:      flightPoint("CGK", mustParseWithLocation("2025-12-15T06:00:00", "Asia/Jakarta")),
				Arrival:        flightPoint("DPS", mustParseWithLocation("2025-12-15T08:00:00", "Asia/Makassar")),
				Duration:       domain.Duration{TotalMinutes: 60, Formatted: "1h"},
				Stops:          0,
				Price:          domain.Price{Amount: 1000000, Currency: "IDR", Display: "IDR 1,000,000"},
				AvailableSeats: 20,
				CabinClass:     domain.CabinEconomy,
				Aircraft:       ptr("A320"),
				Amenities:      []string{"wifi", "meal"},
				Baggage:        domain.Baggage{CarryOn: "7 kg", Checked: "20 kg"},
			},
		},
		{
			name: "invalid timezone in departure time",
			input: lionFlight{
				ID:       "JT999",
				Carrier:  lionCarrier{Name: "Lion Air", IATA: "JT"},
				Route:    lionRoute{From: lionAirport{Code: "CGK"}, To: lionAirport{Code: "DPS"}},
				Schedule: lionSchedule{Departure: "2025-12-15T06:00:00", DepTimezone: "Invalid/Zone", Arrival: "2025-12-15T08:00:00", ArrTimezone: "Asia/Makassar"},
				IsDirect: true,
				Pricing:  lionPricing{Total: 500000, Currency: "IDR", FareType: "ECONOMY"},
				Services: lionServices{Baggage: lionBaggage{Cabin: "7 kg", Hold: "20 kg"}},
			},
			wantErr: true,
		},
		{
			name: "invalid arrival time format",
			input: lionFlight{
				ID:       "JT999",
				Carrier:  lionCarrier{Name: "Lion Air", IATA: "JT"},
				Route:    lionRoute{From: lionAirport{Code: "CGK"}, To: lionAirport{Code: "DPS"}},
				Schedule: lionSchedule{Departure: "2025-12-15T06:00:00", DepTimezone: "Asia/Jakarta", Arrival: "invalid-time", ArrTimezone: "Asia/Makassar"},
				IsDirect: true,
				Pricing:  lionPricing{Total: 500000, Currency: "IDR", FareType: "ECONOMY"},
				Services: lionServices{Baggage: lionBaggage{Cabin: "7 kg", Hold: "20 kg"}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.normalize(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("normalize() mismatch\ngot:  %+v\nwant: %+v", got, tt.want)
			}
		})
	}
}
