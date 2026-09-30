package owner

import (
	"hotel-booking/utils"
	"testing"
	"time"
)

func TestNewRoomType_RejectsZeroOccupancy(t *testing.T) {
	_, err := NewRoomType("prop1", "Deluxe", 0, Money(3000), nil, 3)
	if err == nil {
		t.Fatal("expected error for zero maxOccupancy, got nil")
	}
}

func TestNewRoomType_RejectsZeroRoomCount(t *testing.T) {
	_, err := NewRoomType("prop1", "Deluxe", 2, Money(3000), nil, 0)
	if err == nil {
		t.Fatal("expected error for zero roomCount, got nil")
	}
}

func TestNewRoomType_ValidInputSucceeds(t *testing.T) {
	rt, err := NewRoomType("prop1", "Deluxe", 2, Money(3000), []Amenity{AmenityAirConditioning}, 3)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rt.ID == "" {
		t.Error("expected an auto-generated, non-empty ID")
	}
	if !rt.FitsGuests(2) {
		t.Error("expected room to fit 2 guests")
	}
	if rt.FitsGuests(3) {
		t.Error("expected room to reject 3 guests, max is 2")
	}
	if !rt.HasAmenity(AmenityAirConditioning) {
		t.Error("expected room to have AC")
	}
	if rt.HasAmenity(AmenityPool) {
		t.Error("did not expect room to have pool")
	}
}

func TestNewProperty_RejectsOutOfRangeStarRating(t *testing.T) {
	_, err := NewProperty("o1", "Taj Residency", "Bengaluru", "MG Road", 6, nil)
	if err == nil {
		t.Fatal("expected error for star rating 6, got nil")
	}
	_, err = NewProperty("o1", "Taj Residency", "Bengaluru", "MG Road", 0, nil)
	if err == nil {
		t.Fatal("expected error for star rating 0, got nil")
	}
}

func TestProperty_AddRoomType(t *testing.T) {
	p, err := NewProperty("o1", "Taj Residency", "Bengaluru", "MG Road", 4, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	p.AddRoomType("rt1")
	p.AddRoomType("rt2")
	if len(p.RoomTypeIDs) != 2 {
		t.Errorf("expected 2 room type IDs, got %d", len(p.RoomTypeIDs))
	}
}

func TestOwner_IsSingleProperty(t *testing.T) {
	o, err := NewOwner("Vee's Hotels")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if o.IsSingleProperty() {
		t.Error("expected owner with zero properties to not be single-property")
	}
	o.AddProperty("p1")
	if !o.IsSingleProperty() {
		t.Error("expected owner with exactly one property to be single-property")
	}
	o.AddProperty("p2")
	if o.IsSingleProperty() {
		t.Error("expected owner with two properties to not be single-property")
	}
}

func TestDateRange_HalfOpen_Nights(t *testing.T) {
	checkIn := time.Date(2027, 3, 5, 0, 0, 0, 0, time.UTC)
	checkOut := time.Date(2027, 3, 8, 0, 0, 0, 0, time.UTC)
	dr, err := NewDateRange(checkIn, checkOut)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	nights := dr.Nights()
	if len(nights) != 3 {
		t.Fatalf("expected 3 nights (5, 6, 7), got %d", len(nights))
	}
	for _, n := range nights {
		if n.Day() == 8 {
			t.Error("checkout day must not be included as a booked night")
		}
	}
}

func TestDateRange_RejectsNonPositiveRange(t *testing.T) {
	same := time.Date(2027, 3, 5, 0, 0, 0, 0, time.UTC)
	_, err := NewDateRange(same, same)
	if err == nil {
		t.Fatal("expected error when check-out equals check-in, got nil")
	}
}

func TestDateRange_Overlaps(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2027, 3, d, 0, 0, 0, 0, time.UTC) }
	a, _ := NewDateRange(day(5), day(8))
	b, _ := NewDateRange(day(7), day(10))
	if !a.Overlaps(b) {
		t.Error("expected overlapping ranges to report true")
	}
	// a's checkout (8) equals b's check-in (8) - half-open, so no shared night.
	c, _ := NewDateRange(day(8), day(10))
	if a.Overlaps(c) {
		t.Error("expected back-to-back ranges sharing only the checkout/checkin day to not overlap")
	}
}

func TestMoney_Percentage_RoundsToNearestRupee(t *testing.T) {
	m := Money(999)
	got := m.Percentage(80) // 799.2 -> rounds to 799
	if got != Money(799) {
		t.Errorf("expected 799, got %d", got.Rupees())
	}
}

func TestNewID_NoCollisionsAcrossManyCalls(t *testing.T) {
	seen := make(map[string]bool)
	const n = 10000
	for i := 0; i < n; i++ {
		id := utils.NewID()
		if seen[id] {
			t.Fatalf("collision detected after %d generated ids", i)
		}
		seen[id] = true
	}
}
