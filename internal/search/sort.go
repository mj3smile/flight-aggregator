package search

import (
	"slices"

	"github.com/mj3smile/flight-aggregator/internal/domain"
)

func applySort(flights []domain.Flight, sortBy string) []domain.Flight {
	if sortBy == "" {
		sortBy = "best_value"
	}

	if sortBy == "best_value" {
		flights = scoreFlights(flights)
	}

	slices.SortStableFunc(flights, func(a, b domain.Flight) int {
		switch sortBy {
		case "price_asc":
			return a.Price.Amount - b.Price.Amount
		case "price_desc":
			return b.Price.Amount - a.Price.Amount
		case "duration_asc":
			return a.Duration.TotalMinutes - b.Duration.TotalMinutes
		case "duration_desc":
			return b.Duration.TotalMinutes - a.Duration.TotalMinutes
		case "departure_asc":
			return int(a.Departure.Timestamp - b.Departure.Timestamp)
		case "departure_desc":
			return int(b.Departure.Timestamp - a.Departure.Timestamp)
		case "arrival_asc":
			return int(a.Arrival.Timestamp - b.Arrival.Timestamp)
		case "arrival_desc":
			return int(b.Arrival.Timestamp - a.Arrival.Timestamp)
		case "best_value":
			if a.Score != nil && b.Score != nil {
				if *a.Score < *b.Score {
					return -1
				}
				if *a.Score > *b.Score {
					return 1
				}
			}
			return 0
		default:
			return a.Price.Amount - b.Price.Amount
		}
	})

	return flights
}
