package owner

import "fmt"

type RoomType struct {
	ID           string
	PropertyID   string
	Name         string
	MaxOccupancy int
	BasePrice    Money
	Amenities    []Amenity
	RoomCount    int
}


func NewRoomType(propertyID, name string, maxOccupancy int, basePrice Money, amenities []Amenity, roomCount int) (*RoomType, error) {
	if propertyID == "" {
		return nil, fmt.Errorf("roomtype: propertyID is required")
	}
	if name == "" {
		return nil, fmt.Errorf("roomtype: name is required")
	}
	if maxOccupancy <= 0 {
		return nil, fmt.Errorf("roomtype: maxOccupancy must be positive, got %d", maxOccupancy)
	}
	if roomCount <= 0 {
		return nil, fmt.Errorf("roomtype: roomCount must be positive, got %d", roomCount)
	}
	return &RoomType{
		ID:           newID(),
		PropertyID:   propertyID,
		Name:         name,
		MaxOccupancy: maxOccupancy,
		BasePrice:    basePrice,
		Amenities:    amenities,
		RoomCount:    roomCount,
	}, nil
}

func (r *RoomType) HasAmenity(a Amenity) bool {
	return HasAmenity(r.Amenities, a)
}

func (r *RoomType) FitsGuests(guests int) bool {
	return guests > 0 && guests <= r.MaxOccupancy
}
