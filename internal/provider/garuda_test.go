package provider

import (
	"reflect"
	"testing"

	"github.com/mj3smile/flight-aggregator/internal/domain"
)

func TestGarudaNormalize(t *testing.T) {
	p := &GarudaProvider{}

	tests := []struct {
		name    string
		input   garudaFlight
		want    domain.Flight
		wantErr bool
	}{
		{
			name: "direct flight",
			input: garudaFlight{
				FlightID:    "GA400",
				Airline:     "Garuda Indonesia",
				AirlineCode: "GA",
				Departure:   garudaEndpoint{Airport: "CGK", City: "Jakarta", Time: "2025-12-15T06:00:00+07:00"},
				Arrival:     garudaEndpoint{Airport: "DPS", City: "Denpasar", Time: "2025-12-15T08:50:00+08:00"},
				DurationMin: 110,
				Stops:       0,
				Aircraft:    "Boeing 737-800",
				Price:       garudaPrice{Amount: 1250000, Currency: "IDR"},
				Seats:       28,
				FareClass:   "economy",
				Baggage:     garudaBaggage{CarryOn: 1, Checked: 2},
				Amenities:   []string{"wifi", "meal"},
			},
			want: domain.Flight{
				ID:             "GA400_Garuda Indonesia",
				Provider:       "Garuda Indonesia",
				Airline:        domain.Airline{Name: "Garuda Indonesia", Code: "GA"},
				FlightNumber:   "GA400",
				Departure:      flightPoint("CGK", mustParseRFC3339("2025-12-15T06:00:00+07:00")),
				Arrival:        flightPoint("DPS", mustParseRFC3339("2025-12-15T08:50:00+08:00")),
				Duration:       domain.Duration{TotalMinutes: 110, Formatted: "1h 50m"},
				Stops:          0,
				Price:          domain.Price{Amount: 1250000, Currency: "IDR", Display: "IDR 1,250,000"},
				AvailableSeats: 28,
				CabinClass:     "economy",
				Aircraft:       ptr("Boeing 737-800"),
				Amenities:      []string{"wifi", "meal"},
				Baggage:        domain.Baggage{CarryOn: "1 piece", Checked: "2 pieces"},
			},
		},
		{
			name: "with segments overrides arrival and stops",
			input: garudaFlight{
				FlightID:    "GA315",
				Airline:     "Garuda Indonesia",
				AirlineCode: "GA",
				Departure:   garudaEndpoint{Airport: "CGK", City: "Jakarta", Time: "2025-12-15T14:00:00+07:00"},
				Arrival:     garudaEndpoint{Airport: "SUB", City: "Surabaya", Time: "2025-12-15T15:30:00+07:00"},
				DurationMin: 90,
				Stops:       0,
				Aircraft:    "Boeing 737",
				Price:       garudaPrice{Amount: 1850000, Currency: "IDR"},
				Seats:       22,
				FareClass:   "economy",
				Baggage:     garudaBaggage{CarryOn: 2, Checked: 2},
				Segments: []garudaSegment{
					{
						FlightNumber: "GA315",
						Departure:    garudaSegPoint{Airport: "CGK", Time: "2025-12-15T14:00:00+07:00"},
						Arrival:      garudaSegPoint{Airport: "SUB", Time: "2025-12-15T15:30:00+07:00"},
						DurationMin:  90,
					},
					{
						FlightNumber: "GA332",
						Departure:    garudaSegPoint{Airport: "SUB", Time: "2025-12-15T17:15:00+07:00"},
						Arrival:      garudaSegPoint{Airport: "DPS", Time: "2025-12-15T18:45:00+08:00"},
						DurationMin:  90,
						LayoverMin:   105,
					},
				},
			},
			want: domain.Flight{
				ID:             "GA315_Garuda Indonesia",
				Provider:       "Garuda Indonesia",
				Airline:        domain.Airline{Name: "Garuda Indonesia", Code: "GA"},
				FlightNumber:   "GA315",
				Departure:      flightPoint("CGK", mustParseRFC3339("2025-12-15T14:00:00+07:00")),
				Arrival:        flightPoint("DPS", mustParseRFC3339("2025-12-15T18:45:00+08:00")),
				Duration:       domain.Duration{TotalMinutes: 225, Formatted: "3h 45m"},
				Stops:          1,
				Price:          domain.Price{Amount: 1850000, Currency: "IDR", Display: "IDR 1,850,000"},
				AvailableSeats: 22,
				CabinClass:     "economy",
				Aircraft:       ptr("Boeing 737"),
				Amenities:      []string{},
				Baggage:        domain.Baggage{CarryOn: "2 pieces", Checked: "2 pieces"},
			},
		},
		{
			name: "nil amenities becomes empty slice",
			input: garudaFlight{
				FlightID:    "GA100",
				Airline:     "Garuda Indonesia",
				AirlineCode: "GA",
				Departure:   garudaEndpoint{Airport: "CGK", Time: "2025-12-15T06:00:00+07:00"},
				Arrival:     garudaEndpoint{Airport: "DPS", Time: "2025-12-15T08:00:00+08:00"},
				Aircraft:    "A320",
				Price:       garudaPrice{Amount: 1000000, Currency: "IDR"},
				Seats:       10,
				Baggage:     garudaBaggage{CarryOn: 1, Checked: 1},
				Amenities:   nil,
			},
			want: domain.Flight{
				ID:             "GA100_Garuda Indonesia",
				Provider:       "Garuda Indonesia",
				Airline:        domain.Airline{Name: "Garuda Indonesia", Code: "GA"},
				FlightNumber:   "GA100",
				Departure:      flightPoint("CGK", mustParseRFC3339("2025-12-15T06:00:00+07:00")),
				Arrival:        flightPoint("DPS", mustParseRFC3339("2025-12-15T08:00:00+08:00")),
				Duration:       domain.Duration{TotalMinutes: 60, Formatted: "1h"},
				Stops:          0,
				Price:          domain.Price{Amount: 1000000, Currency: "IDR", Display: "IDR 1,000,000"},
				AvailableSeats: 10,
				Aircraft:       ptr("A320"),
				Amenities:      []string{},
				Baggage:        domain.Baggage{CarryOn: "1 piece", Checked: "1 piece"},
			},
		},
		{
			name: "invalid departure time",
			input: garudaFlight{
				FlightID:  "GA999",
				Airline:   "Garuda Indonesia",
				Departure: garudaEndpoint{Airport: "CGK", Time: "invalid"},
				Arrival:   garudaEndpoint{Airport: "DPS", Time: "2025-12-15T08:00:00+08:00"},
			},
			wantErr: true,
		},
		{
			name: "invalid arrival time",
			input: garudaFlight{
				FlightID:  "GA999",
				Airline:   "Garuda Indonesia",
				Departure: garudaEndpoint{Airport: "CGK", Time: "2025-12-15T06:00:00+07:00"},
				Arrival:   garudaEndpoint{Airport: "DPS", Time: "not-a-time"},
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
