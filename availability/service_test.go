package availability

import (
	"sync"
	"testing"
	"time"

	"hotel-booking/owner"
)

func dateRange(t *testing.T, checkIn, checkOut int) owner.DateRange {
	t.Helper()
	dr, err := owner.NewDateRange(
		time.Date(2027, 3, checkIn, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 3, checkOut, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("fixture date range setup failed: %v", err)
	}
	return dr
}

func TestIsAvailable_UnregisteredRoomType_ReturnsError(t *testing.T) {
	s := NewService()
	_, err := s.IsAvailable("unknown", dateRange(t, 5, 8))
	if err == nil {
		t.Fatal("expected error for unregistered room type, got nil")
	}
}

func TestReserve_SucceedsWhenRoomsAvailable(t *testing.T) {
	s := NewService()
	s.Register("deluxe", 3)

	ok, err := s.Reserve("deluxe", dateRange(t, 5, 8))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !ok {
		t.Fatal("expected reservation to succeed with rooms available")
	}
}

func TestReserve_DecrementsOnlyTouchedNights(t *testing.T) {
	s := NewService()
	s.Register("deluxe", 3)

	// Book March 5-8 (nights 5, 6, 7).
	if _, err := s.Reserve("deluxe", dateRange(t, 5, 8)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	available, err := s.IsAvailable("deluxe", dateRange(t, 20, 21))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !available {
		t.Error("expected March 20 to remain fully available - it was never booked")
	}
}

func TestReserve_FailsWhenNoRoomsRemain(t *testing.T) {
	s := NewService()
	s.Register("deluxe", 1) // only 1 room

	ok, err := s.Reserve("deluxe", dateRange(t, 6, 7))
	if err != nil || !ok {
		t.Fatalf("expected first reservation to succeed, got ok=%v err=%v", ok, err)
	}

	ok, err = s.Reserve("deluxe", dateRange(t, 6, 7))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected second reservation for the same last room to fail")
	}
}

func TestReserve_MultiNightRange_AllOrNothing(t *testing.T) {
	s := NewService()
	s.Register("deluxe", 1)

	// Take the only room on March 7 specifically.
	if _, err := s.Reserve("deluxe", dateRange(t, 7, 8)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ok, err := s.Reserve("deluxe", dateRange(t, 5, 8))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected reservation to fail when any single night in the range is full")
	}

	available, err := s.IsAvailable("deluxe", dateRange(t, 5, 6))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !available {
		t.Error("expected March 5 to remain available - the failed multi-night request must not have partially decremented it")
	}
}

func TestRelease_MakesRoomAvailableAgain(t *testing.T) {
	s := NewService()
	s.Register("deluxe", 1)

	if _, err := s.Reserve("deluxe", dateRange(t, 6, 7)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := s.Release("deluxe", dateRange(t, 6, 7)); err != nil {
		t.Fatalf("unexpected error on release: %v", err)
	}

	ok, err := s.Reserve("deluxe", dateRange(t, 6, 7))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected the room to be bookable again after release")
	}
}

func TestReserve_ConcurrentRequestsForLastRoom(t *testing.T) {
	s := NewService()
	s.Register("deluxe", 1)

	const attempts = 100
	var wg sync.WaitGroup
	results := make([]bool, attempts)

	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ok, err := s.Reserve("deluxe", dateRange(t, 6, 7))
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			results[idx] = ok
		}(i)
	}
	wg.Wait()

	successes := 0
	for _, ok := range results {
		if ok {
			successes++
		}
	}
	if successes != 1 {
		t.Errorf("expected exactly 1 successful reservation out of %d concurrent attempts, got %d", attempts, successes)
	}
}
