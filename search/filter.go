package search

import (
	"strings"

	"hotel-booking/availability"
	"hotel-booking/owner"
)

type Filter interface {
	Matches(p *owner.Property) bool
}

// --- Shallow filters: read Property's own fields directly ---

type CityFilter struct {
	City string
}

func (f CityFilter) Matches(p *owner.Property) bool {
	return strings.EqualFold(p.City, f.City)
}

type StarRatingFilter struct {
	MinRating int
}

func (f StarRatingFilter) Matches(p *owner.Property) bool {
	return p.StarRating >= f.MinRating
}

type PropertyAmenityFilter struct {
	Amenity owner.Amenity
}

func (f PropertyAmenityFilter) Matches(p *owner.Property) bool {
	return p.HasAmenity(f.Amenity)
}

// --- Deep filters: traverse into this property's room types ---

type PriceRangeFilter struct {
	RoomTypes owner.RoomTypeRepository
	Min, Max  owner.Money
}

func (f PriceRangeFilter) Matches(p *owner.Property) bool {
	roomTypes, err := f.RoomTypes.FindByPropertyID(p.ID)
	if err != nil {
		return false
	}
	for _, rt := range roomTypes {
		if rt.BasePrice >= f.Min && rt.BasePrice <= f.Max {
			return true
		}
	}
	return false
}

type RoomAmenityFilter struct {
	RoomTypes owner.RoomTypeRepository
	Amenity   owner.Amenity
}

func (f RoomAmenityFilter) Matches(p *owner.Property) bool {
	roomTypes, err := f.RoomTypes.FindByPropertyID(p.ID)
	if err != nil {
		return false
	}
	for _, rt := range roomTypes {
		if rt.HasAmenity(f.Amenity) {
			return true
		}
	}
	return false
}

type AvailabilityFilter struct {
	RoomTypes    owner.RoomTypeRepository
	Availability availability.Service
	DateRange    owner.DateRange
	GuestCount   int
}

func (f AvailabilityFilter) Matches(p *owner.Property) bool {
	roomTypes, err := f.RoomTypes.FindByPropertyID(p.ID)
	if err != nil {
		return false
	}
	for _, rt := range roomTypes {
		if !rt.FitsGuests(f.GuestCount) {
			continue
		}
		ok, err := f.Availability.IsAvailable(rt.ID, f.DateRange)
		if err == nil && ok {
			return true
		}
	}
	return false
}
