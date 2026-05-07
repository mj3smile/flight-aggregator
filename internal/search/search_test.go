package search

import (
	"math"
	"reflect"
	"testing"

	"github.com/mj3smile/flight-aggregator/internal/domain"
)

func sampleFlights() []domain.Flight {
	return []domain.Flight{
		{
			ID: "F1", FlightNumber: "GA400",
			Airline:   domain.Airline{Name: "Garuda Indonesia", Code: "GA"},
			Departure: domain.FlightPoint{Airport: "CGK", Datetime: "2025-12-15T06:00:00+07:00", Timestamp: 1734220800},
			Arrival:   domain.FlightPoint{Airport: "DPS", Datetime: "2025-12-15T08:50:00+08:00", Timestamp: 1734231000},
			Duration:  domain.Duration{TotalMinutes: 110, Formatted: "1h 50m"},
			Stops:     0,
			Price:     domain.Price{Amount: 1250000, Currency: "IDR"},
		},
		{
			ID: "F2", FlightNumber: "JT740",
			Airline:   domain.Airline{Name: "Lion Air", Code: "JT"},
			Departure: domain.FlightPoint{Airport: "CGK", Datetime: "2025-12-15T05:30:00+07:00", Timestamp: 1734219000},
			Arrival:   domain.FlightPoint{Airport: "DPS", Datetime: "2025-12-15T08:15:00+08:00", Timestamp: 1734228900},
			Duration:  domain.Duration{TotalMinutes: 105, Formatted: "1h 45m"},
			Stops:     0,
			Price:     domain.Price{Amount: 950000, Currency: "IDR"},
		},
		{
			ID: "F3", FlightNumber: "QZ7250",
			Airline:   domain.Airline{Name: "AirAsia", Code: "QZ"},
			Departure: domain.FlightPoint{Airport: "CGK", Datetime: "2025-12-15T15:15:00+07:00", Timestamp: 1734254100},
			Arrival:   domain.FlightPoint{Airport: "DPS", Datetime: "2025-12-15T20:35:00+08:00", Timestamp: 1734267300},
			Duration:  domain.Duration{TotalMinutes: 260, Formatted: "4h 20m"},
			Stops:     1,
			Price:     domain.Price{Amount: 485000, Currency: "IDR"},
		},
		{
			ID: "F4", FlightNumber: "GA315",
			Airline:   domain.Airline{Name: "Garuda Indonesia", Code: "GA"},
			Departure: domain.FlightPoint{Airport: "CGK", Datetime: "2025-12-15T14:00:00+07:00", Timestamp: 1734249600},
			Arrival:   domain.FlightPoint{Airport: "DPS", Datetime: "2025-12-15T18:45:00+08:00", Timestamp: 1734266700},
			Duration:  domain.Duration{TotalMinutes: 285, Formatted: "4h 45m"},
			Stops:     2,
			Price:     domain.Price{Amount: 1850000, Currency: "IDR"},
		},
	}
}

func flightIDs(flights []domain.Flight) []string {
	ids := make([]string, len(flights))
	for i, f := range flights {
		ids[i] = f.ID
	}
	return ids
}

func TestApplyFilters(t *testing.T) {
	tests := []struct {
		name    string
		filters *domain.Filters
		wantIDs []string
	}{
		{
			name:    "nil filters returns all",
			filters: nil,
			wantIDs: []string{"F1", "F2", "F3", "F4"},
		},
		{
			name:    "empty filters returns all",
			filters: &domain.Filters{},
			wantIDs: []string{"F1", "F2", "F3", "F4"},
		},
		{
			name:    "min price",
			filters: &domain.Filters{MinPrice: intPtr(1000000)},
			wantIDs: []string{"F1", "F4"},
		},
		{
			name:    "max price",
			filters: &domain.Filters{MaxPrice: intPtr(1000000)},
			wantIDs: []string{"F2", "F3"},
		},
		{
			name:    "price range",
			filters: &domain.Filters{MinPrice: intPtr(900000), MaxPrice: intPtr(1300000)},
			wantIDs: []string{"F1", "F2"},
		},
		{
			name:    "max stops 0 direct only",
			filters: &domain.Filters{MaxStops: intPtr(0)},
			wantIDs: []string{"F1", "F2"},
		},
		{
			name:    "max stops 1",
			filters: &domain.Filters{MaxStops: intPtr(1)},
			wantIDs: []string{"F1", "F2", "F3"},
		},
		{
			name:    "max duration",
			filters: &domain.Filters{MaxDuration: intPtr(120)},
			wantIDs: []string{"F1", "F2"},
		},
		{
			name:    "airline by code",
			filters: &domain.Filters{Airlines: []string{"GA"}},
			wantIDs: []string{"F1", "F4"},
		},
		{
			name:    "airline by name case insensitive",
			filters: &domain.Filters{Airlines: []string{"lion air"}},
			wantIDs: []string{"F2"},
		},
		{
			name:    "multiple airlines",
			filters: &domain.Filters{Airlines: []string{"GA", "QZ"}},
			wantIDs: []string{"F1", "F3", "F4"},
		},
		{
			name:    "departure after",
			filters: &domain.Filters{DepartureAfter: strPtr("06:00")},
			wantIDs: []string{"F1", "F3", "F4"},
		},
		{
			name:    "departure before",
			filters: &domain.Filters{DepartureBefore: strPtr("14:30")},
			wantIDs: []string{"F1", "F2", "F4"},
		},
		{
			name:    "departure time window",
			filters: &domain.Filters{DepartureAfter: strPtr("06:00"), DepartureBefore: strPtr("15:00")},
			wantIDs: []string{"F1", "F4"},
		},
		{
			name:    "arrival after",
			filters: &domain.Filters{ArrivalAfter: strPtr("09:00")},
			wantIDs: []string{"F3", "F4"},
		},
		{
			name:    "arrival before",
			filters: &domain.Filters{ArrivalBefore: strPtr("09:00")},
			wantIDs: []string{"F1", "F2"},
		},
		{
			name:    "arrival time window",
			filters: &domain.Filters{ArrivalAfter: strPtr("08:00"), ArrivalBefore: strPtr("19:00")},
			wantIDs: []string{"F1", "F2", "F4"},
		},
		{
			name:    "combined max price and direct only",
			filters: &domain.Filters{MaxPrice: intPtr(1000000), MaxStops: intPtr(0)},
			wantIDs: []string{"F2"},
		},
		{
			name:    "filters exclude all",
			filters: &domain.Filters{MaxPrice: intPtr(100000)},
			wantIDs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyFilters(sampleFlights(), tt.filters)
			gotIDs := flightIDs(result)
			if !reflect.DeepEqual(gotIDs, tt.wantIDs) {
				t.Errorf("got IDs %v, want %v", gotIDs, tt.wantIDs)
			}
		})
	}
}

func TestApplySort(t *testing.T) {
	tests := []struct {
		name    string
		sortBy  string
		wantIDs []string
	}{
		{
			name:    "price ascending",
			sortBy:  "price_asc",
			wantIDs: []string{"F3", "F2", "F1", "F4"},
		},
		{
			name:    "price descending",
			sortBy:  "price_desc",
			wantIDs: []string{"F4", "F1", "F2", "F3"},
		},
		{
			name:    "duration ascending",
			sortBy:  "duration_asc",
			wantIDs: []string{"F2", "F1", "F3", "F4"},
		},
		{
			name:    "duration descending",
			sortBy:  "duration_desc",
			wantIDs: []string{"F4", "F3", "F1", "F2"},
		},
		{
			name:    "departure ascending",
			sortBy:  "departure_asc",
			wantIDs: []string{"F2", "F1", "F4", "F3"},
		},
		{
			name:    "departure descending",
			sortBy:  "departure_desc",
			wantIDs: []string{"F3", "F4", "F1", "F2"},
		},
		{
			name:    "arrival ascending",
			sortBy:  "arrival_asc",
			wantIDs: []string{"F2", "F1", "F4", "F3"},
		},
		{
			name:    "arrival descending",
			sortBy:  "arrival_desc",
			wantIDs: []string{"F3", "F4", "F1", "F2"},
		},
		{
			name:    "best value",
			sortBy:  "best_value",
			wantIDs: []string{"F2", "F1", "F3", "F4"},
		},
		{
			name:    "empty defaults to best value",
			sortBy:  "",
			wantIDs: []string{"F2", "F1", "F3", "F4"},
		},
		{
			name:    "unknown defaults to price ascending",
			sortBy:  "unknown_sort",
			wantIDs: []string{"F3", "F2", "F1", "F4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applySort(sampleFlights(), tt.sortBy)
			gotIDs := flightIDs(result)
			if !reflect.DeepEqual(gotIDs, tt.wantIDs) {
				t.Errorf("got IDs %v, want %v", gotIDs, tt.wantIDs)
			}
		})
	}
}

func TestApplySort_BestValueSetsScores(t *testing.T) {
	result := applySort(sampleFlights(), "best_value")
	for _, f := range result {
		if f.Score == nil {
			t.Fatalf("score should not be nil for flight %s", f.ID)
		}
	}
	for i := 0; i < len(result)-1; i++ {
		if *result[i].Score > *result[i+1].Score {
			t.Errorf("flight %s (score %.4f) should have score <= flight %s (score %.4f)",
				result[i].ID, *result[i].Score, result[i+1].ID, *result[i+1].Score)
		}
	}
}

func TestScoreFlights(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		result := scoreFlights(nil)
		if len(result) != 0 {
			t.Errorf("expected empty, got %d", len(result))
		}
	})

	t.Run("single direct flight scores 0", func(t *testing.T) {
		flights := []domain.Flight{{
			ID: "X1", Stops: 0,
			Price:    domain.Price{Amount: 500000},
			Duration: domain.Duration{TotalMinutes: 100},
		}}
		result := scoreFlights(flights)
		if result[0].Score == nil {
			t.Fatal("score should not be nil")
		}
		if *result[0].Score != 0 {
			t.Errorf("single direct flight score = %f, want 0", *result[0].Score)
		}
	})

	t.Run("single 1-stop flight scores 0.1", func(t *testing.T) {
		flights := []domain.Flight{{
			ID: "X1", Stops: 1,
			Price:    domain.Price{Amount: 500000},
			Duration: domain.Duration{TotalMinutes: 100},
		}}
		result := scoreFlights(flights)
		if result[0].Score == nil {
			t.Fatal("score should not be nil")
		}
		if !approxEqual(*result[0].Score, 0.1) {
			t.Errorf("single 1-stop flight score = %f, want 0.1", *result[0].Score)
		}
	})

	t.Run("single 2-stop flight scores 0.2", func(t *testing.T) {
		flights := []domain.Flight{{
			ID: "X1", Stops: 2,
			Price:    domain.Price{Amount: 500000},
			Duration: domain.Duration{TotalMinutes: 100},
		}}
		result := scoreFlights(flights)
		if result[0].Score == nil {
			t.Fatal("score should not be nil")
		}
		if !approxEqual(*result[0].Score, 0.2) {
			t.Errorf("single 2-stop flight score = %f, want 0.2", *result[0].Score)
		}
	})

	t.Run("all scores in 0 to 1", func(t *testing.T) {
		result := scoreFlights(sampleFlights())
		for _, f := range result {
			if f.Score == nil {
				t.Fatalf("score nil for %s", f.ID)
			}
			if *f.Score < 0 || *f.Score > 1 {
				t.Errorf("score %f out of [0,1] for %s", *f.Score, f.ID)
			}
		}
	})

	t.Run("worst flight scores 1.0", func(t *testing.T) {
		result := scoreFlights(sampleFlights())
		var f4 domain.Flight
		for _, f := range result {
			if f.ID == "F4" {
				f4 = f
				break
			}
		}
		if f4.Score == nil {
			t.Fatal("F4 score should not be nil")
		}
		if !approxEqual(*f4.Score, 1.0) {
			t.Errorf("F4 (most expensive, longest, 2 stops) score = %f, want 1.0", *f4.Score)
		}
	})

	t.Run("same price and duration differ only by stops", func(t *testing.T) {
		flights := []domain.Flight{
			{ID: "A", Stops: 0, Price: domain.Price{Amount: 500000}, Duration: domain.Duration{TotalMinutes: 100}},
			{ID: "B", Stops: 1, Price: domain.Price{Amount: 500000}, Duration: domain.Duration{TotalMinutes: 100}},
			{ID: "C", Stops: 2, Price: domain.Price{Amount: 500000}, Duration: domain.Duration{TotalMinutes: 100}},
		}
		result := scoreFlights(flights)

		if !approxEqual(*result[0].Score, 0.0) {
			t.Errorf("direct flight score = %f, want 0.0", *result[0].Score)
		}
		if !approxEqual(*result[1].Score, 0.1) {
			t.Errorf("1-stop flight score = %f, want 0.1", *result[1].Score)
		}
		if !approxEqual(*result[2].Score, 0.2) {
			t.Errorf("2-stop flight score = %f, want 0.2", *result[2].Score)
		}
	})
}

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func intPtr(v int) *int {
	return &v
}

func strPtr(v string) *string {
	return &v
}
