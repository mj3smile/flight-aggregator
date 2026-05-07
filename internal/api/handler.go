package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/mj3smile/flight-aggregator/internal/aggregator"
	"github.com/mj3smile/flight-aggregator/internal/domain"
)

type Handler struct {
	aggregator *aggregator.Aggregator
}

func NewHandler(agg *aggregator.Aggregator) *Handler {
	return &Handler{aggregator: agg}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/flights/search", h.searchFlights)
	mux.HandleFunc("GET /api/v1/health", h.health)
}

func (h *Handler) searchFlights(w http.ResponseWriter, r *http.Request) {
	var req domain.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if err := validateSearchRequest(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.aggregator.Search(r.Context(), req)
	if err != nil {
		slog.Error("search failed", "error", err)
		writeError(w, http.StatusServiceUnavailable, "all flight providers are currently unavailable, please try again later")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

const dateFormat = "2006-01-02"

func validateSearchRequest(req domain.SearchRequest) error {
	if req.Origin == "" {
		return errors.New("origin is required")
	}

	if !domain.ValidAirport(req.Origin) {
		return errors.New("origin is not a valid airport code")
	}

	if req.Destination == "" {
		return errors.New("destination is required")
	}

	if !domain.ValidAirport(req.Destination) {
		return errors.New("destination is not a valid airport code")
	}

	if req.Origin == req.Destination {
		return errors.New("origin and destination must be different")
	}

	if req.DepartureDate == "" {
		return errors.New("departureDate is required")
	}

	_, err := time.Parse(dateFormat, req.DepartureDate)
	if err != nil {
		return errors.New("departureDate must be in YYYY-MM-DD format")
	}

	// uncomment this in production to make sure the departure date is not in the past
	//now := time.Now()
	//today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	//if depDate.Before(today) {
	//	return errors.New("departureDate must not be in the past")
	//}

	// not implemented
	//if req.ReturnDate != nil && *req.ReturnDate != "" {
	//	retDate, err := time.Parse(dateFormat, *req.ReturnDate)
	//	if err != nil {
	//		return errors.New("returnDate must be in YYYY-MM-DD format")
	//	}
	//	if retDate.Before(depDate) {
	//		return errors.New("returnDate must not be before departureDate")
	//	}
	//}

	if req.Passengers <= 0 {
		return errors.New("passengers must be at least 1")
	}

	if req.CabinClass != "" {
		switch req.CabinClass {
		case domain.CabinEconomy, domain.CabinBusiness, domain.CabinFirst:
		default:
			return fmt.Errorf("cabinClass must be one of: %s, %s, %s", domain.CabinEconomy, domain.CabinBusiness, domain.CabinFirst)
		}
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
