package search

import "github.com/mj3smile/flight-aggregator/internal/domain"

// scoreFlights computes a "best value" score for each flight.
//
// score formula (lower is better):
//
//	0.5 * normalized_price + 0.3 * normalized_duration + 0.2 * stops_penalty
//
// normalization maps each metric to [0, 1] relative to the min/max in the result set.
// stops penalty: 0 for direct, 0.5 for 1 stop, 1.0 for 2+ stops.
func scoreFlights(flights []domain.Flight) []domain.Flight {
	if len(flights) == 0 {
		return flights
	}

	minPrice, maxPrice := flights[0].Price.Amount, flights[0].Price.Amount
	minDur, maxDur := flights[0].Duration.TotalMinutes, flights[0].Duration.TotalMinutes

	for _, f := range flights[1:] {
		if f.Price.Amount < minPrice {
			minPrice = f.Price.Amount
		}
		if f.Price.Amount > maxPrice {
			maxPrice = f.Price.Amount
		}
		if f.Duration.TotalMinutes < minDur {
			minDur = f.Duration.TotalMinutes
		}
		if f.Duration.TotalMinutes > maxDur {
			maxDur = f.Duration.TotalMinutes
		}
	}

	priceRange := float64(maxPrice - minPrice)
	durRange := float64(maxDur - minDur)

	for i := range flights {
		var normPrice, normDur float64
		if priceRange > 0 {
			normPrice = float64(flights[i].Price.Amount-minPrice) / priceRange
		}
		if durRange > 0 {
			normDur = float64(flights[i].Duration.TotalMinutes-minDur) / durRange
		}

		var stopsPenalty float64
		switch {
		case flights[i].Stops == 0:
			stopsPenalty = 0
		case flights[i].Stops == 1:
			stopsPenalty = 0.5
		default:
			stopsPenalty = 1.0
		}

		score := 0.5*normPrice + 0.3*normDur + 0.2*stopsPenalty
		flights[i].Score = &score
	}

	return flights
}
