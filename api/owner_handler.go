package api

import (
	"encoding/json"
	"net/http"

	"hotel-booking/owner"
)

func (s *Server) createOwner(w http.ResponseWriter, r *http.Request) {
	var req createOwnerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, err)
		return
	}

	o, err := owner.NewOwner(req.Name)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.ownerRepo.Save(o); err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, ownerResponse{ID: o.ID, Name: o.Name})
}

func (s *Server) addProperty(w http.ResponseWriter, r *http.Request) {
	ownerID := r.PathValue("ownerID")

	o, err := s.ownerRepo.FindByID(ownerID)
	if err != nil {
		writeError(w, err)
		return
	}

	var req addPropertyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, err)
		return
	}

	p, err := owner.NewProperty(ownerID, req.Name, req.City, req.Locality, req.StarRating, parseAmenities(req.Amenities))
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.propertyRepo.Save(p); err != nil {
		writeError(w, err)
		return
	}

	o.AddProperty(p.ID)
	if err := s.ownerRepo.Save(o); err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, propertyResponse{
		ID: p.ID, OwnerID: p.OwnerID, Name: p.Name, City: p.City,
		Locality: p.Locality, StarRating: p.StarRating,
		Amenities: amenitiesToStrings(p.Amenities),
	})
}

func (s *Server) addRoomType(w http.ResponseWriter, r *http.Request) {
	propertyID := r.PathValue("propertyID")

	p, err := s.propertyRepo.FindByID(propertyID)
	if err != nil {
		writeError(w, err)
		return
	}

	var req addRoomTypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, err)
		return
	}

	rt, err := owner.NewRoomType(propertyID, req.Name, req.MaxOccupancy, owner.Money(req.BasePrice), parseAmenities(req.Amenities), req.RoomCount)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.roomTypeRepo.Save(rt); err != nil {
		writeError(w, err)
		return
	}

	p.AddRoomType(rt.ID)
	if err := s.propertyRepo.Save(p); err != nil {
		writeError(w, err)
		return
	}

	s.availability.Register(rt.ID, rt.RoomCount)

	writeJSON(w, http.StatusCreated, roomTypeResponse{
		ID: rt.ID, PropertyID: rt.PropertyID, Name: rt.Name,
		MaxOccupancy: rt.MaxOccupancy, BasePrice: rt.BasePrice.Rupees(),
		Amenities: amenitiesToStrings(rt.Amenities), RoomCount: rt.RoomCount,
	})
}
