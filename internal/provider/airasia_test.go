package provider

import (
	"reflect"
	"testing"

	"github.com/mj3smile/flight-aggregator/internal/domain"
)

func TestAirAsiaNormalize(t *testing.T) {
	p := &AirAsiaProvider{}

	tests := []struct {
		name    string
		input   airAsiaFlight
		want    domain.Flight
		wantErr bool
	}{
		{
			name: "direct flight",
			input: airAsiaFlight{
				FlightCode:   "QZ520",
				Airline:      "AirAsia",
				FromAirport:  "CGK",
				ToAirport:    "DPS",
				DepartTime:   "2025-12-15T04:45:00+07:00",
				ArriveTime:   "2025-12-15T07:25:00+08:00",
				DurationHrs:  1.67,
				DirectFlight: true,
				PriceIDR:     650000,
				Seats:        67,
				CabinClass:   "economy",
				BaggageNote:  "Cabin baggage only, checked bags additional fee",
			},
			want: domain.Flight{
				ID:             "QZ520_AirAsia",
				Provider:       "AirAsia",
				Airline:        domain.Airline{Name: "AirAsia", Code: "QZ"},
				FlightNumber:   "QZ520",
				Departure:      flightPoint("CGK", mustParseRFC3339("2025-12-15T04:45:00+07:00")),
				Arrival:        flightPoint("DPS", mustParseRFC3339("2025-12-15T07:25:00+08:00")),
				Duration:       domain.Duration{TotalMinutes: 100, Formatted: "1h 40m"},
				Stops:          0,
				Price:          domain.Price{Amount: 650000, Currency: "IDR", Display: "IDR 650,000"},
				AvailableSeats: 67,
				CabinClass:     "economy",
				Aircraft:       nil,
				Amenities:      []string{},
				Baggage:        domain.Baggage{CarryOn: "Cabin baggage only", Checked: "checked bags additional fee"},
			},
		},
		{
			name: "connecting flight with stops array",
			input: airAsiaFlight{
				FlightCode:   "QZ7250",
				Airline:      "AirAsia",
				FromAirport:  "CGK",
				ToAirport:    "DPS",
				DepartTime:   "2025-12-15T15:15:00+07:00",
				ArriveTime:   "2025-12-15T20:35:00+08:00",
				DurationHrs:  4.33,
				DirectFlight: false,
				PriceIDR:     485000,
				Seats:        88,
				CabinClass:   "economy",
				BaggageNote:  "Cabin baggage only, checked bags additional fee",
				Stops:        []airAsiaStop{{Airport: "SOC", WaitTime: 95}},
			},
			want: domain.Flight{
				ID:             "QZ7250_AirAsia",
				Provider:       "AirAsia",
				Airline:        domain.Airline{Name: "AirAsia", Code: "QZ"},
				FlightNumber:   "QZ7250",
				Departure:      flightPoint("CGK", mustParseRFC3339("2025-12-15T15:15:00+07:00")),
				Arrival:        flightPoint("DPS", mustParseRFC3339("2025-12-15T20:35:00+08:00")),
				Duration:       domain.Duration{TotalMinutes: 260, Formatted: "4h 20m"},
				Stops:          1,
				Price:          domain.Price{Amount: 485000, Currency: "IDR", Display: "IDR 485,000"},
				AvailableSeats: 88,
				CabinClass:     "economy",
				Aircraft:       nil,
				Amenities:      []string{},
				Baggage:        domain.Baggage{CarryOn: "Cabin baggage only", Checked: "checked bags additional fee"},
			},
		},
		{
			name: "non-direct without stops array defaults to 1 stop",
			input: airAsiaFlight{
				FlightCode:   "QZ999",
				Airline:      "AirAsia",
				FromAirport:  "CGK",
				ToAirport:    "DPS",
				DepartTime:   "2025-12-15T10:00:00+07:00",
				ArriveTime:   "2025-12-15T14:00:00+08:00",
				DirectFlight: false,
				PriceIDR:     400000,
				Seats:        30,
				CabinClass:   "economy",
				BaggageNote:  "Cabin only",
				Stops:        nil,
			},
			want: domain.Flight{
				ID:             "QZ999_AirAsia",
				Provider:       "AirAsia",
				Airline:        domain.Airline{Name: "AirAsia", Code: "QZ"},
				FlightNumber:   "QZ999",
				Departure:      flightPoint("CGK", mustParseRFC3339("2025-12-15T10:00:00+07:00")),
				Arrival:        flightPoint("DPS", mustParseRFC3339("2025-12-15T14:00:00+08:00")),
				Duration:       domain.Duration{TotalMinutes: 180, Formatted: "3h"},
				Stops:          1,
				Price:          domain.Price{Amount: 400000, Currency: "IDR", Display: "IDR 400,000"},
				AvailableSeats: 30,
				CabinClass:     "economy",
				Aircraft:       nil,
				Amenities:      []string{},
				Baggage:        domain.Baggage{CarryOn: "Cabin only", Checked: ""},
			},
		},
		{
			name: "multiple stops",
			input: airAsiaFlight{
				FlightCode:   "QZ800",
				Airline:      "AirAsia",
				FromAirport:  "CGK",
				ToAirport:    "DPS",
				DepartTime:   "2025-12-15T08:00:00+07:00",
				ArriveTime:   "2025-12-15T16:00:00+08:00",
				DirectFlight: false,
				PriceIDR:     350000,
				Seats:        100,
				CabinClass:   "economy",
				BaggageNote:  "Cabin baggage only, checked bags extra",
				Stops:        []airAsiaStop{{Airport: "SUB", WaitTime: 60}, {Airport: "UPG", WaitTime: 45}},
			},
			want: domain.Flight{
				ID:             "QZ800_AirAsia",
				Provider:       "AirAsia",
				Airline:        domain.Airline{Name: "AirAsia", Code: "QZ"},
				FlightNumber:   "QZ800",
				Departure:      flightPoint("CGK", mustParseRFC3339("2025-12-15T08:00:00+07:00")),
				Arrival:        flightPoint("DPS", mustParseRFC3339("2025-12-15T16:00:00+08:00")),
				Duration:       domain.Duration{TotalMinutes: 420, Formatted: "7h"},
				Stops:          2,
				Price:          domain.Price{Amount: 350000, Currency: "IDR", Display: "IDR 350,000"},
				AvailableSeats: 100,
				CabinClass:     "economy",
				Aircraft:       nil,
				Amenities:      []string{},
				Baggage:        domain.Baggage{CarryOn: "Cabin baggage only", Checked: "checked bags extra"},
			},
		},
		{
			name: "airline code extracted from flight code prefix",
			input: airAsiaFlight{
				FlightCode:   "D7123",
				Airline:      "AirAsia X",
				FromAirport:  "CGK",
				ToAirport:    "DPS",
				DepartTime:   "2025-12-15T10:00:00+07:00",
				ArriveTime:   "2025-12-15T12:00:00+08:00",
				DirectFlight: true,
				PriceIDR:     800000,
				Seats:        40,
				CabinClass:   "economy",
				BaggageNote:  "Cabin only, no checked",
			},
			want: domain.Flight{
				ID:             "D7123_AirAsia X",
				Provider:       "AirAsia X",
				Airline:        domain.Airline{Name: "AirAsia X", Code: "D7"},
				FlightNumber:   "D7123",
				Departure:      flightPoint("CGK", mustParseRFC3339("2025-12-15T10:00:00+07:00")),
				Arrival:        flightPoint("DPS", mustParseRFC3339("2025-12-15T12:00:00+08:00")),
				Duration:       domain.Duration{TotalMinutes: 60, Formatted: "1h"},
				Stops:          0,
				Price:          domain.Price{Amount: 800000, Currency: "IDR", Display: "IDR 800,000"},
				AvailableSeats: 40,
				CabinClass:     "economy",
				Aircraft:       nil,
				Amenities:      []string{},
				Baggage:        domain.Baggage{CarryOn: "Cabin only", Checked: "no checked"},
			},
		},
		{
			name: "invalid departure time",
			input: airAsiaFlight{
				FlightCode:   "QZ000",
				Airline:      "AirAsia",
				FromAirport:  "CGK",
				ToAirport:    "DPS",
				DepartTime:   "not-a-date",
				ArriveTime:   "2025-12-15T12:00:00+08:00",
				DirectFlight: true,
				PriceIDR:     500000,
				Seats:        50,
				CabinClass:   "economy",
				BaggageNote:  "Cabin only",
			},
			wantErr: true,
		},
		{
			name: "invalid arrival time",
			input: airAsiaFlight{
				FlightCode:   "QZ000",
				Airline:      "AirAsia",
				FromAirport:  "CGK",
				ToAirport:    "DPS",
				DepartTime:   "2025-12-15T10:00:00+07:00",
				ArriveTime:   "invalid",
				DirectFlight: true,
				PriceIDR:     500000,
				Seats:        50,
				CabinClass:   "economy",
				BaggageNote:  "Cabin only",
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
