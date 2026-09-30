package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"hotel-booking/booking"
	"hotel-booking/owner"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, owner.ErrNotFound) || errors.Is(err, booking.ErrNotFound) {
		status = http.StatusNotFound
	}
	writeJSON(w, status, errorResponse{Error: err.Error()})
}

func parseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}

func parseAmenities(raw []string) []owner.Amenity {
	amenities := make([]owner.Amenity, 0, len(raw))
	for _, r := range raw {
		amenities = append(amenities, owner.Amenity(r))
	}
	return amenities
}

func amenitiesToStrings(amenities []owner.Amenity) []string {
	out := make([]string, 0, len(amenities))
	for _, a := range amenities {
		out = append(out, string(a))
	}
	return out
}
