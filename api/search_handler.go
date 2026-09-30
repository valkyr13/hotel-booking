package api

import (
	"net/http"
	"strconv"

	"hotel-booking/owner"
	"hotel-booking/search"
)

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var filters []search.Filter

	if city := q.Get("city"); city != "" {
		filters = append(filters, search.CityFilter{City: city})
	}
	if v := q.Get("minStars"); v != "" {
		minStars, err := strconv.Atoi(v)
		if err != nil {
			writeError(w, err)
			return
		}
		filters = append(filters, search.StarRatingFilter{MinRating: minStars})
	}
	if v := q.Get("amenity"); v != "" {
		filters = append(filters, search.PropertyAmenityFilter{Amenity: owner.Amenity(v)})
	}
	if minP, maxP := q.Get("minPrice"), q.Get("maxPrice"); minP != "" && maxP != "" {
		min, err := strconv.ParseInt(minP, 10, 64)
		if err != nil {
			writeError(w, err)
			return
		}
		max, err := strconv.ParseInt(maxP, 10, 64)
		if err != nil {
			writeError(w, err)
			return
		}
		filters = append(filters, search.PriceRangeFilter{RoomTypes: s.roomTypeRepo, Min: owner.Money(min), Max: owner.Money(max)})
	}
	if checkIn, checkOut := q.Get("checkIn"), q.Get("checkOut"); checkIn != "" && checkOut != "" {
		in, err := parseDate(checkIn)
		if err != nil {
			writeError(w, err)
			return
		}
		out, err := parseDate(checkOut)
		if err != nil {
			writeError(w, err)
			return
		}
		dr, err := owner.NewDateRange(in, out)
		if err != nil {
			writeError(w, err)
			return
		}
		guests := 1
		if v := q.Get("guests"); v != "" {
			guests, err = strconv.Atoi(v)
			if err != nil {
				writeError(w, err)
				return
			}
		}
		filters = append(filters, search.AvailabilityFilter{
			RoomTypes: s.roomTypeRepo, Availability: s.availability, DateRange: dr, GuestCount: guests,
		})
	}

	results, err := s.searchSvc.Search(filters...)
	if err != nil {
		writeError(w, err)
		return
	}

	out := make([]searchResultResponse, 0, len(results))
	for _, p := range results {
		out = append(out, searchResultResponse{
			ID: p.ID, Name: p.Name, City: p.City, Locality: p.Locality,
			StarRating: p.StarRating, Amenities: amenitiesToStrings(p.Amenities),
		})
	}
	writeJSON(w, http.StatusOK, out)
}
