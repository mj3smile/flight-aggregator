package search

import (
	"strings"

	"github.com/mj3smile/flight-aggregator/internal/domain"
)

func Apply(flights []domain.Flight, filters *domain.Filters, sortBy string) []domain.Flight {
	result := applyFilters(flights, filters)
	result = applySort(result, sortBy)
	return result
}

func applyFilters(flights []domain.Flight, f *domain.Filters) []domain.Flight {
	if f == nil {
		return flights
	}

	result := make([]domain.Flight, 0, len(flights))
	for _, flight := range flights {
		if !matchesFilters(flight, f) {
			continue
		}
		result = append(result, flight)
	}
	return result
}

func matchesFilters(f domain.Flight, filters *domain.Filters) bool {
	if filters.MinPrice != nil && f.Price.Amount < *filters.MinPrice {
		return false
	}
	if filters.MaxPrice != nil && f.Price.Amount > *filters.MaxPrice {
		return false
	}
	if filters.MaxStops != nil && f.Stops > *filters.MaxStops {
		return false
	}
	if filters.MaxDuration != nil && f.Duration.TotalMinutes > *filters.MaxDuration {
		return false
	}

	if len(filters.Airlines) > 0 {
		matched := false
		for _, a := range filters.Airlines {
			if strings.EqualFold(f.Airline.Name, a) || strings.EqualFold(f.Airline.Code, a) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	if filters.DepartureAfter != nil {
		depHHMM := f.Departure.Datetime[11:16]
		if depHHMM < *filters.DepartureAfter {
			return false
		}
	}
	if filters.DepartureBefore != nil {
		depHHMM := f.Departure.Datetime[11:16]
		if depHHMM > *filters.DepartureBefore {
			return false
		}
	}
	if filters.ArrivalAfter != nil {
		arrHHMM := f.Arrival.Datetime[11:16]
		if arrHHMM < *filters.ArrivalAfter {
			return false
		}
	}
	if filters.ArrivalBefore != nil {
		arrHHMM := f.Arrival.Datetime[11:16]
		if arrHHMM > *filters.ArrivalBefore {
			return false
		}
	}

	return true
}
