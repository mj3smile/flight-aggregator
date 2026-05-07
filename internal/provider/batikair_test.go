package provider

import (
	"reflect"
	"testing"

	"github.com/mj3smile/flight-aggregator/internal/domain"
)

func TestBatikAirNormalize(t *testing.T) {
	p := &BatikAirProvider{}

	tests := []struct {
		name    string
		input   batikFlight
		want    domain.Flight
		wantErr bool
	}{
		{
			name: "direct economy flight",
			input: batikFlight{
				FlightNumber: "ID6514",
				AirlineName:  "Batik Air",
				AirlineIATA:  "ID",
				Origin:       "CGK",
				Destination:  "DPS",
				DepDateTime:  "2025-12-15T07:15:00+0700",
				ArrDateTime:  "2025-12-15T10:00:00+0800",
				TravelTime:   "1h 45m",
				Stops:        0,
				Fare:         batikFare{BasePrice: 980000, Taxes: 120000, Total: 1100000, Currency: "IDR", Class: "Y"},
				Seats:        32,
				Aircraft:     "Airbus A320",
				BaggageInfo:  "7kg cabin, 20kg checked",
				Services:     []string{"Snack", "Beverage"},
			},
			want: domain.Flight{
				ID:             "ID6514_Batik Air",
				Provider:       "Batik Air",
				Airline:        domain.Airline{Name: "Batik Air", Code: "ID"},
				FlightNumber:   "ID6514",
				Departure:      flightPoint("CGK", mustParseCompactOffset("2025-12-15T07:15:00+0700")),
				Arrival:        flightPoint("DPS", mustParseCompactOffset("2025-12-15T10:00:00+0800")),
				Duration:       domain.Duration{TotalMinutes: 105, Formatted: "1h 45m"},
				Stops:          0,
				Price:          domain.Price{Amount: 1100000, Currency: "IDR", Display: "IDR 1,100,000"},
				AvailableSeats: 32,
				CabinClass:     domain.CabinEconomy,
				Aircraft:       ptr("Airbus A320"),
				Amenities:      []string{"snack", "beverage"},
				Baggage:        domain.Baggage{CarryOn: "7kg cabin", Checked: "20kg checked"},
			},
		},
		{
			name: "business class",
			input: batikFlight{
				FlightNumber: "ID100",
				AirlineName:  "Batik Air",
				AirlineIATA:  "ID",
				Origin:       "CGK",
				Destination:  "DPS",
				DepDateTime:  "2025-12-15T07:00:00+0700",
				ArrDateTime:  "2025-12-15T09:00:00+0800",
				Fare:         batikFare{Total: 3000000, Currency: "IDR", Class: "C"},
				Seats:        4,
				Aircraft:     "A330",
				BaggageInfo:  "10kg cabin, 30kg checked",
				Services:     []string{"Meal", "Lounge"},
			},
			want: domain.Flight{
				ID:             "ID100_Batik Air",
				Provider:       "Batik Air",
				Airline:        domain.Airline{Name: "Batik Air", Code: "ID"},
				FlightNumber:   "ID100",
				Departure:      flightPoint("CGK", mustParseCompactOffset("2025-12-15T07:00:00+0700")),
				Arrival:        flightPoint("DPS", mustParseCompactOffset("2025-12-15T09:00:00+0800")),
				Duration:       domain.Duration{TotalMinutes: 60, Formatted: "1h"},
				Stops:          0,
				Price:          domain.Price{Amount: 3000000, Currency: "IDR", Display: "IDR 3,000,000"},
				AvailableSeats: 4,
				CabinClass:     domain.CabinBusiness,
				Aircraft:       ptr("A330"),
				Amenities:      []string{"meal", "lounge"},
				Baggage:        domain.Baggage{CarryOn: "10kg cabin", Checked: "30kg checked"},
			},
		},
		{
			name: "first class",
			input: batikFlight{
				FlightNumber: "ID200",
				AirlineName:  "Batik Air",
				AirlineIATA:  "ID",
				Origin:       "CGK",
				Destination:  "DPS",
				DepDateTime:  "2025-12-15T07:00:00+0700",
				ArrDateTime:  "2025-12-15T09:00:00+0800",
				Fare:         batikFare{Total: 5000000, Currency: "IDR", Class: "F"},
				Seats:        2,
				Aircraft:     "A380",
				BaggageInfo:  "15kg cabin, 40kg checked",
				Services:     []string{"Full Meal"},
			},
			want: domain.Flight{
				ID:             "ID200_Batik Air",
				Provider:       "Batik Air",
				Airline:        domain.Airline{Name: "Batik Air", Code: "ID"},
				FlightNumber:   "ID200",
				Departure:      flightPoint("CGK", mustParseCompactOffset("2025-12-15T07:00:00+0700")),
				Arrival:        flightPoint("DPS", mustParseCompactOffset("2025-12-15T09:00:00+0800")),
				Duration:       domain.Duration{TotalMinutes: 60, Formatted: "1h"},
				Stops:          0,
				Price:          domain.Price{Amount: 5000000, Currency: "IDR", Display: "IDR 5,000,000"},
				AvailableSeats: 2,
				CabinClass:     domain.CabinFirst,
				Aircraft:       ptr("A380"),
				Amenities:      []string{"full meal"},
				Baggage:        domain.Baggage{CarryOn: "15kg cabin", Checked: "40kg checked"},
			},
		},
		{
			name: "nil services becomes empty amenities",
			input: batikFlight{
				FlightNumber: "ID300",
				AirlineName:  "Batik Air",
				AirlineIATA:  "ID",
				Origin:       "CGK",
				Destination:  "DPS",
				DepDateTime:  "2025-12-15T07:00:00+0700",
				ArrDateTime:  "2025-12-15T09:00:00+0800",
				Fare:         batikFare{Total: 800000, Currency: "IDR", Class: "Y"},
				Seats:        50,
				Aircraft:     "737",
				BaggageInfo:  "7kg cabin, 20kg checked",
				Services:     nil,
			},
			want: domain.Flight{
				ID:             "ID300_Batik Air",
				Provider:       "Batik Air",
				Airline:        domain.Airline{Name: "Batik Air", Code: "ID"},
				FlightNumber:   "ID300",
				Departure:      flightPoint("CGK", mustParseCompactOffset("2025-12-15T07:00:00+0700")),
				Arrival:        flightPoint("DPS", mustParseCompactOffset("2025-12-15T09:00:00+0800")),
				Duration:       domain.Duration{TotalMinutes: 60, Formatted: "1h"},
				Stops:          0,
				Price:          domain.Price{Amount: 800000, Currency: "IDR", Display: "IDR 800,000"},
				AvailableSeats: 50,
				CabinClass:     domain.CabinEconomy,
				Aircraft:       ptr("737"),
				Amenities:      []string{},
				Baggage:        domain.Baggage{CarryOn: "7kg cabin", Checked: "20kg checked"},
			},
		},
		{
			name: "baggage info without comma",
			input: batikFlight{
				FlightNumber: "ID400",
				AirlineName:  "Batik Air",
				AirlineIATA:  "ID",
				Origin:       "CGK",
				Destination:  "DPS",
				DepDateTime:  "2025-12-15T07:00:00+0700",
				ArrDateTime:  "2025-12-15T09:00:00+0800",
				Fare:         batikFare{Total: 700000, Currency: "IDR", Class: "Y"},
				Seats:        60,
				Aircraft:     "737",
				BaggageInfo:  "cabin only",
			},
			want: domain.Flight{
				ID:             "ID400_Batik Air",
				Provider:       "Batik Air",
				Airline:        domain.Airline{Name: "Batik Air", Code: "ID"},
				FlightNumber:   "ID400",
				Departure:      flightPoint("CGK", mustParseCompactOffset("2025-12-15T07:00:00+0700")),
				Arrival:        flightPoint("DPS", mustParseCompactOffset("2025-12-15T09:00:00+0800")),
				Duration:       domain.Duration{TotalMinutes: 60, Formatted: "1h"},
				Stops:          0,
				Price:          domain.Price{Amount: 700000, Currency: "IDR", Display: "IDR 700,000"},
				AvailableSeats: 60,
				CabinClass:     domain.CabinEconomy,
				Aircraft:       ptr("737"),
				Amenities:      []string{},
				Baggage:        domain.Baggage{CarryOn: "cabin only", Checked: ""},
			},
		},
		{
			name: "with stops",
			input: batikFlight{
				FlightNumber: "ID7042",
				AirlineName:  "Batik Air",
				AirlineIATA:  "ID",
				Origin:       "CGK",
				Destination:  "DPS",
				DepDateTime:  "2025-12-15T18:45:00+0700",
				ArrDateTime:  "2025-12-15T23:50:00+0800",
				TravelTime:   "3h 5m",
				Stops:        1,
				Fare:         batikFare{Total: 950000, Currency: "IDR", Class: "Y"},
				Seats:        41,
				Aircraft:     "Airbus A320",
				BaggageInfo:  "7kg cabin, 20kg checked",
				Services:     []string{"Snack"},
			},
			want: domain.Flight{
				ID:             "ID7042_Batik Air",
				Provider:       "Batik Air",
				Airline:        domain.Airline{Name: "Batik Air", Code: "ID"},
				FlightNumber:   "ID7042",
				Departure:      flightPoint("CGK", mustParseCompactOffset("2025-12-15T18:45:00+0700")),
				Arrival:        flightPoint("DPS", mustParseCompactOffset("2025-12-15T23:50:00+0800")),
				Duration:       domain.Duration{TotalMinutes: 245, Formatted: "4h 5m"},
				Stops:          1,
				Price:          domain.Price{Amount: 950000, Currency: "IDR", Display: "IDR 950,000"},
				AvailableSeats: 41,
				CabinClass:     domain.CabinEconomy,
				Aircraft:       ptr("Airbus A320"),
				Amenities:      []string{"snack"},
				Baggage:        domain.Baggage{CarryOn: "7kg cabin", Checked: "20kg checked"},
			},
		},
		{
			name: "invalid departure time",
			input: batikFlight{
				FlightNumber: "ID999",
				AirlineName:  "Batik Air",
				AirlineIATA:  "ID",
				Origin:       "CGK",
				Destination:  "DPS",
				DepDateTime:  "bad-time",
				ArrDateTime:  "2025-12-15T09:00:00+0800",
				Fare:         batikFare{Total: 1000000, Currency: "IDR", Class: "Y"},
				BaggageInfo:  "7kg cabin, 20kg checked",
			},
			wantErr: true,
		},
		{
			name: "invalid arrival time",
			input: batikFlight{
				FlightNumber: "ID999",
				AirlineName:  "Batik Air",
				AirlineIATA:  "ID",
				Origin:       "CGK",
				Destination:  "DPS",
				DepDateTime:  "2025-12-15T09:00:00+0700",
				ArrDateTime:  "bad-time",
				Fare:         batikFare{Total: 1000000, Currency: "IDR", Class: "Y"},
				BaggageInfo:  "7kg cabin, 20kg checked",
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
