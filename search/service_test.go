package search

import (
	"testing"
	"time"

	"hotel-booking/availability"
	"hotel-booking/owner"
)

// --- Shallow filter tests ---

func TestCityFilter(t *testing.T) {
	p, _ := owner.NewProperty("o1", "Taj Residency", "Bengaluru", "MG Road", 4, nil)

	if !(CityFilter{City: "bengaluru"}).Matches(p) {
		t.Error("expected case-insensitive city match to succeed")
	}
	if (CityFilter{City: "Mumbai"}).Matches(p) {
		t.Error("expected non-matching city to fail")
	}
}

func TestStarRatingFilter(t *testing.T) {
	p, _ := owner.NewProperty("o1", "Taj Residency", "Bengaluru", "MG Road", 4, nil)

	if !(StarRatingFilter{MinRating: 4}).Matches(p) {
		t.Error("expected property at exactly the minimum rating to match")
	}
	if !(StarRatingFilter{MinRating: 3}).Matches(p) {
		t.Error("expected property above the minimum rating to match")
	}
	if (StarRatingFilter{MinRating: 5}).Matches(p) {
		t.Error("expected property below the minimum rating to fail")
	}
}

func TestPropertyAmenityFilter(t *testing.T) {
	p, _ := owner.NewProperty("o1", "Taj Residency", "Bengaluru", "MG Road", 4, []owner.Amenity{owner.AmenityPool})

	if !(PropertyAmenityFilter{Amenity: owner.AmenityPool}).Matches(p) {
		t.Error("expected property with pool to match")
	}
	if (PropertyAmenityFilter{Amenity: owner.AmenityGym}).Matches(p) {
		t.Error("expected property without gym to fail")
	}
}

// --- Deep filter tests ---

func TestPriceRangeFilter_MatchesIfAnyRoomTypeInRange(t *testing.T) {
	repo := owner.NewInMemoryRoomTypeRepository()
	p, _ := owner.NewProperty("o1", "Taj Residency", "Bengaluru", "MG Road", 4, nil)
	cheap, _ := owner.NewRoomType(p.ID, "Single", 1, owner.Money(1500), nil, 2)
	expensive, _ := owner.NewRoomType(p.ID, "Suite", 4, owner.Money(9000), nil, 1)
	repo.Save(cheap)
	repo.Save(expensive)

	// Range only covers the cheap room type - property should still match.
	filter := PriceRangeFilter{RoomTypes: repo, Min: owner.Money(1000), Max: owner.Money(2000)}
	if !filter.Matches(p) {
		t.Error("expected property to match when at least one room type falls in the price range")
	}

	// Range covers neither room type.
	filter = PriceRangeFilter{RoomTypes: repo, Min: owner.Money(3000), Max: owner.Money(5000)}
	if filter.Matches(p) {
		t.Error("expected property to fail when no room type falls in the price range")
	}
}

func TestRoomAmenityFilter_MatchesIfAnyRoomTypeHasAmenity(t *testing.T) {
	repo := owner.NewInMemoryRoomTypeRepository()
	p, _ := owner.NewProperty("o1", "Taj Residency", "Bengaluru", "MG Road", 4, nil)
	noAC, _ := owner.NewRoomType(p.ID, "Single", 1, owner.Money(1500), nil, 2)
	withAC, _ := owner.NewRoomType(p.ID, "Deluxe", 2, owner.Money(3000), []owner.Amenity{owner.AmenityAirConditioning}, 1)
	repo.Save(noAC)
	repo.Save(withAC)

	filter := RoomAmenityFilter{RoomTypes: repo, Amenity: owner.AmenityAirConditioning}
	if !filter.Matches(p) {
		t.Error("expected property to match when at least one room type has the amenity")
	}

	filter = RoomAmenityFilter{RoomTypes: repo, Amenity: owner.AmenityTV}
	if filter.Matches(p) {
		t.Error("expected property to fail when no room type has the amenity")
	}
}

func TestAvailabilityFilter_MatchesWhenAvailableAndFits(t *testing.T) {
	roomTypeRepo := owner.NewInMemoryRoomTypeRepository()
	p, _ := owner.NewProperty("o1", "Taj Residency", "Bengaluru", "MG Road", 4, nil)
	rt, _ := owner.NewRoomType(p.ID, "Deluxe", 2, owner.Money(3000), nil, 3)
	roomTypeRepo.Save(rt)

	avail := availability.NewService()
	avail.Register(rt.ID, 3)

	dr, _ := owner.NewDateRange(
		time.Date(2027, 3, 5, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 3, 8, 0, 0, 0, 0, time.UTC),
	)

	filter := AvailabilityFilter{RoomTypes: roomTypeRepo, Availability: avail, DateRange: dr, GuestCount: 2}
	if !filter.Matches(p) {
		t.Error("expected property to match when a room type is available and fits the guest count")
	}
}

func TestAvailabilityFilter_NoMatchWhenGuestCountExceedsEveryRoomType(t *testing.T) {
	roomTypeRepo := owner.NewInMemoryRoomTypeRepository()
	p, _ := owner.NewProperty("o1", "Taj Residency", "Bengaluru", "MG Road", 4, nil)
	rt, _ := owner.NewRoomType(p.ID, "Deluxe", 2, owner.Money(3000), nil, 3) // max 2 guests
	roomTypeRepo.Save(rt)

	avail := availability.NewService()
	avail.Register(rt.ID, 3)

	dr, _ := owner.NewDateRange(
		time.Date(2027, 3, 5, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 3, 8, 0, 0, 0, 0, time.UTC),
	)

	filter := AvailabilityFilter{RoomTypes: roomTypeRepo, Availability: avail, DateRange: dr, GuestCount: 5}
	if filter.Matches(p) {
		t.Error("expected property to fail when no room type fits the guest count, regardless of availability")
	}
}

func TestAvailabilityFilter_NoMatchWhenFullyBooked(t *testing.T) {
	roomTypeRepo := owner.NewInMemoryRoomTypeRepository()
	p, _ := owner.NewProperty("o1", "Taj Residency", "Bengaluru", "MG Road", 4, nil)
	rt, _ := owner.NewRoomType(p.ID, "Deluxe", 2, owner.Money(3000), nil, 1) // only 1 room
	roomTypeRepo.Save(rt)

	avail := availability.NewService()
	avail.Register(rt.ID, 1)

	dr, _ := owner.NewDateRange(
		time.Date(2027, 3, 5, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 3, 8, 0, 0, 0, 0, time.UTC),
	)
	avail.Reserve(rt.ID, dr) // take the only room

	filter := AvailabilityFilter{RoomTypes: roomTypeRepo, Availability: avail, DateRange: dr, GuestCount: 2}
	if filter.Matches(p) {
		t.Error("expected property to fail when its only room type is fully booked for the requested dates")
	}
}

// --- Search orchestration tests ---

func TestSearch_ChainsMultipleFilters(t *testing.T) {
	propertyRepo := owner.NewInMemoryPropertyRepository()
	roomTypeRepo := owner.NewInMemoryRoomTypeRepository()

	// Property A: Bengaluru, 4-star, has a cheap room type.
	pA, _ := owner.NewProperty("o1", "Taj Residency", "Bengaluru", "MG Road", 4, nil)
	rtA, _ := owner.NewRoomType(pA.ID, "Single", 1, owner.Money(1500), nil, 3)
	propertyRepo.Save(pA)
	roomTypeRepo.Save(rtA)

	// Property B: Bengaluru, but only 2-star - should be filtered out by rating.
	pB, _ := owner.NewProperty("o1", "Budget Inn", "Bengaluru", "HSR", 2, nil)
	rtB, _ := owner.NewRoomType(pB.ID, "Single", 1, owner.Money(1200), nil, 3)
	propertyRepo.Save(pB)
	roomTypeRepo.Save(rtB)

	// Property C: right city and rating, but in Mumbai - filtered out by city.
	pC, _ := owner.NewProperty("o1", "Grand Mumbai", "Mumbai", "Andheri", 4, nil)
	rtC, _ := owner.NewRoomType(pC.ID, "Single", 1, owner.Money(1500), nil, 3)
	propertyRepo.Save(pC)
	roomTypeRepo.Save(rtC)

	searchSvc := NewService(propertyRepo)
	results, err := searchSvc.Search(
		CityFilter{City: "Bengaluru"},
		StarRatingFilter{MinRating: 4},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected exactly 1 result, got %d", len(results))
	}
	if results[0].ID != pA.ID {
		t.Errorf("expected result to be property A, got %s", results[0].Name)
	}
}

func TestSearch_NoFilters_ReturnsEverything(t *testing.T) {
	propertyRepo := owner.NewInMemoryPropertyRepository()
	p1, _ := owner.NewProperty("o1", "Taj Residency", "Bengaluru", "MG Road", 4, nil)
	p2, _ := owner.NewProperty("o1", "Budget Inn", "Mumbai", "Andheri", 2, nil)
	propertyRepo.Save(p1)
	propertyRepo.Save(p2)

	searchSvc := NewService(propertyRepo)
	results, err := searchSvc.Search()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected all 2 properties with no filters applied, got %d", len(results))
	}
}
