package booking

import (
	"testing"
	"time"

	"hotel-booking/availability"
	"hotel-booking/owner"
)

type testFixture struct {
	service      *Service
	availability availability.Service
	roomTypeID   string
	dateRange    owner.DateRange
}

func newTestFixture(t *testing.T, totalRooms int) *testFixture {
	t.Helper()

	roomTypeRepo := owner.NewInMemoryRoomTypeRepository()
	rt, err := owner.NewRoomType("prop1", "Deluxe", 2, owner.Money(3000), nil, totalRooms)
	if err != nil {
		t.Fatalf("fixture: failed to create room type: %v", err)
	}
	if err := roomTypeRepo.Save(rt); err != nil {
		t.Fatalf("fixture: failed to save room type: %v", err)
	}

	avail := availability.NewService()
	avail.Register(rt.ID, totalRooms)

	dr, err := owner.NewDateRange(
		time.Date(2027, 3, 5, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 3, 8, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("fixture: failed to create date range: %v", err)
	}

	return &testFixture{
		service:      NewService(roomTypeRepo, avail, NewInMemoryRepository()),
		availability: avail,
		roomTypeID:   rt.ID,
		dateRange:    dr,
	}
}

func TestCreateBooking_Success(t *testing.T) {
	f := newTestFixture(t, 3)

	b, err := f.service.CreateBooking("guest1", f.roomTypeID, f.dateRange, 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if b.Status != StatusPendingPayment {
		t.Errorf("expected status PENDING_PAYMENT, got %s", b.Status)
	}
	if b.Amount != owner.Money(9000) { // 3000/night x 3 nights (5,6,7)
		t.Errorf("expected amount 9000, got %d", b.Amount.Rupees())
	}
}

func TestCreateBooking_RejectsWhenGuestCountExceedsOccupancy(t *testing.T) {
	f := newTestFixture(t, 3)

	_, err := f.service.CreateBooking("guest1", f.roomTypeID, f.dateRange, 5) // max is 2
	if err == nil {
		t.Fatal("expected error for guest count exceeding max occupancy, got nil")
	}
}

func TestCreateBooking_RejectsUnknownRoomType(t *testing.T) {
	f := newTestFixture(t, 3)

	_, err := f.service.CreateBooking("guest1", "does-not-exist", f.dateRange, 2)
	if err == nil {
		t.Fatal("expected error for unknown room type, got nil")
	}
}

func TestCreateBooking_RejectsWhenNoRoomsAvailable(t *testing.T) {
	f := newTestFixture(t, 1) // only 1 room

	if _, err := f.service.CreateBooking("guest1", f.roomTypeID, f.dateRange, 2); err != nil {
		t.Fatalf("expected first booking to succeed, got %v", err)
	}

	_, err := f.service.CreateBooking("guest2", f.roomTypeID, f.dateRange, 2)
	if err == nil {
		t.Fatal("expected second booking for the same last room to fail")
	}
}

func TestConfirmPayment_Success(t *testing.T) {
	f := newTestFixture(t, 3)
	b, _ := f.service.CreateBooking("guest1", f.roomTypeID, f.dateRange, 2)

	if err := f.service.ConfirmPayment(b.ID, "txn-fixture-1"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	updated, _ := f.service.bookings.FindByID(b.ID)
	if updated.Status != StatusConfirmed {
		t.Errorf("expected status CONFIRMED, got %s", updated.Status)
	}
	if updated.TransactionID != "txn-fixture-1" {
		t.Errorf("expected transaction ID to be recorded, got %q", updated.TransactionID)
	}
}

func TestFailPayment_Success_AndReleasesInventory(t *testing.T) {
	f := newTestFixture(t, 1) // only 1 room, so release is directly observable
	b, _ := f.service.CreateBooking("guest1", f.roomTypeID, f.dateRange, 2)

	if err := f.service.FailPayment(b.ID); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	updated, _ := f.service.bookings.FindByID(b.ID)
	if updated.Status != StatusFailed {
		t.Errorf("expected status FAILED, got %s", updated.Status)
	}

	ok, err := f.availability.IsAvailable(f.roomTypeID, f.dateRange)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected inventory to be released after payment failure - the room should be bookable again")
	}
}

func TestCancelBooking_Success_AndReleasesInventory(t *testing.T) {
	f := newTestFixture(t, 1)
	b, _ := f.service.CreateBooking("guest1", f.roomTypeID, f.dateRange, 2)
	if err := f.service.ConfirmPayment(b.ID, "txn-fixture-1"); err != nil {
		t.Fatalf("fixture setup: unexpected error confirming: %v", err)
	}

	if err := f.service.CancelBooking(b.ID); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	updated, _ := f.service.bookings.FindByID(b.ID)
	if updated.Status != StatusCancelled {
		t.Errorf("expected status CANCELLED, got %s", updated.Status)
	}

	ok, err := f.availability.IsAvailable(f.roomTypeID, f.dateRange)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected inventory to be released after cancellation - the room should be bookable again")
	}
}

// --- Negative / rejected-transition tests ---
// These verify the system correctly BLOCKS an invalid transition, not
// that it succeeds.

func TestCancelBooking_Rejected_WhenStillPendingPayment(t *testing.T) {
	f := newTestFixture(t, 3)
	b, _ := f.service.CreateBooking("guest1", f.roomTypeID, f.dateRange, 2)

	err := f.service.CancelBooking(b.ID)
	if err == nil {
		t.Fatal("expected cancelling a PENDING_PAYMENT booking to be rejected, got nil error")
	}

	updated, _ := f.service.bookings.FindByID(b.ID)
	if updated.Status != StatusPendingPayment {
		t.Errorf("expected status to remain PENDING_PAYMENT after rejected cancel, got %s", updated.Status)
	}
}

func TestConfirmPayment_Rejected_WhenAlreadyConfirmed(t *testing.T) {
	f := newTestFixture(t, 3)
	b, _ := f.service.CreateBooking("guest1", f.roomTypeID, f.dateRange, 2)
	if err := f.service.ConfirmPayment(b.ID, "txn-fixture-1"); err != nil {
		t.Fatalf("fixture setup: unexpected error: %v", err)
	}

	err := f.service.ConfirmPayment(b.ID, "txn-fixture-1")
	if err == nil {
		t.Fatal("expected confirming an already-CONFIRMED booking to be rejected, got nil error")
	}
}

func TestCancelBooking_Rejected_WhenAlreadyCancelled(t *testing.T) {
	f := newTestFixture(t, 3)
	b, _ := f.service.CreateBooking("guest1", f.roomTypeID, f.dateRange, 2)
	f.service.ConfirmPayment(b.ID, "txn-fixture-1")
	if err := f.service.CancelBooking(b.ID); err != nil {
		t.Fatalf("fixture setup: unexpected error: %v", err)
	}

	err := f.service.CancelBooking(b.ID)
	if err == nil {
		t.Fatal("expected cancelling an already-CANCELLED booking to be rejected, got nil error")
	}
}

func TestFailPayment_Rejected_WhenAlreadyFailed(t *testing.T) {
	f := newTestFixture(t, 3)
	b, _ := f.service.CreateBooking("guest1", f.roomTypeID, f.dateRange, 2)
	if err := f.service.FailPayment(b.ID); err != nil {
		t.Fatalf("fixture setup: unexpected error: %v", err)
	}

	err := f.service.FailPayment(b.ID)
	if err == nil {
		t.Fatal("expected failing an already-FAILED booking to be rejected, got nil error")
	}
}
