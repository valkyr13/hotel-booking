package owner

import (
	
	"fmt"
	"hotel-booking/utils"
)

type Property struct {
	ID          string
	OwnerID     string
	Name        string
	City        string
	Locality    string
	StarRating  int
	Amenities   []Amenity
	RoomTypeIDs []string
}


func NewProperty(ownerID, name, city, locality string, starRating int, amenities []Amenity) (*Property, error) {
	if ownerID == "" {
		return nil, fmt.Errorf("property: ownerID is required")
	}
	if name == "" {
		return nil, fmt.Errorf("property: name is required")
	}
	if city == "" {
		return nil, fmt.Errorf("property: city is required")
	}
	if starRating < 1 || starRating > 5 {
		return nil, fmt.Errorf("property: starRating must be between 1 and 5, got %d", starRating)
	}
	return &Property{
		ID:         utils.NewID(),
		OwnerID:    ownerID,
		Name:       name,
		City:       city,
		Locality:   locality,
		StarRating: starRating,
		Amenities:  amenities,
	}, nil
}


func (p *Property) AddRoomType(roomTypeID string) {
	p.RoomTypeIDs = append(p.RoomTypeIDs, roomTypeID)
}


func (p *Property) HasAmenity(a Amenity) bool {
	return HasAmenity(p.Amenities, a)
}
