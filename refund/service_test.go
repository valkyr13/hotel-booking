package refund

import (
	"testing"
	"time"

	"hotel-booking/availability"
	"hotel-booking/booking"
	"hotel-booking/owner"
)

func bookingWithCheckIn(t *testing.T, checkIn time.Time, amount owner.Money) *booking.Booking {
	t.Helper()
	dr, err := owner.NewDateRange(checkIn, checkIn.AddDate(0, 0, 1))
	if err != nil {
		t.Fatalf("fixture: failed to build date range: %v", err)
	}
	return &booking.Booking{
		ID:        "fixture-booking",
		DateRange: dr,
		Amount:    amount,
		Status:    booking.StatusConfirmed,
	}
}

func TestTieredRefundPolicy_Boundaries(t *testing.T) {
	policy := NewTieredRefundPolicy()
	checkIn := time.Date(2027, 3, 15, 0, 0, 0, 0, time.UTC)
	amount := owner.Money(1000)

	cases := []struct {
		name            string
		daysBeforeCheck int
		wantPct         int
	}{
		{"exactly 10 days out - top tier", 10, 90},
		{"11 days out - still top tier", 11, 90},
		{"exactly 9 days out - just under top tier", 9, 80},
		{"exactly 3 days out - middle tier boundary", 3, 80},
		{"exactly 2 days out - just under middle tier", 2, 50},
		{"exactly 1 day out - lower tier boundary", 1, 50},
		{"same-day (0 days out)", 0, 30},
		{"cancelling after check-in already passed", -1, 30},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cancelDate := checkIn.AddDate(0, 0, -c.daysBeforeCheck)
			b := bookingWithCheckIn(t, checkIn, amount)
			got := policy.CalculateRefund(b, cancelDate)
			want := amount.Percentage(c.wantPct)
			if got != want {
				t.Errorf("expected %v (%d%%), got %v", want, c.wantPct, got)
			}
		})
	}
}

type testFixture struct {
	refundSvc    *Service
	bookingSvc   *booking.Service
	availability availability.Service
	roomTypeID   string
	dateRange    owner.DateRange
}

func newTestFixture(t *testing.T) *testFixture {
	t.Helper()

	roomTypeRepo := owner.NewInMemoryRoomTypeRepository()
	rt, _ := owner.NewRoomType("prop1", "Deluxe", 2, owner.Money(3000), nil, 1)
	roomTypeRepo.Save(rt)

	avail := availability.NewService()
	avail.Register(rt.ID, 1)

	dr, _ := owner.NewDateRange(
		time.Date(2027, 3, 15, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 3, 18, 0, 0, 0, 0, time.UTC),
	)

	bookingSvc := booking.NewService(roomTypeRepo, avail, booking.NewInMemoryRepository())

	return &testFixture{
		refundSvc:    NewService(NewTieredRefundPolicy(), bookingSvc),
		bookingSvc:   bookingSvc,
		availability: avail,
		roomTypeID:   rt.ID,
		dateRange:    dr,
	}
}

func TestService_Cancel_Success_ReturnsCorrectRefundAndReleasesInventory(t *testing.T) {
	f := newTestFixture(t)
	b, err := f.bookingSvc.CreateBooking("guest1", f.roomTypeID, f.dateRange, 2)
	if err != nil {
		t.Fatalf("fixture setup: unexpected error: %v", err)
	}
	if err := f.bookingSvc.ConfirmPayment(b.ID, "txn-1"); err != nil {
		t.Fatalf("fixture setup: unexpected error confirming: %v", err)
	}

	// 15 days before the March 15 check-in - top tier, 90%.
	cancelDate := time.Date(2027, 2, 28, 0, 0, 0, 0, time.UTC)
	refund, err := f.refundSvc.Cancel(b.ID, cancelDate)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	wantRefund := b.Amount.Percentage(90)
	if refund != wantRefund {
		t.Errorf("expected refund %v, got %v", wantRefund, refund)
	}

	updated, _ := f.bookingSvc.FindByID(b.ID)
	if updated.Status != booking.StatusCancelled {
		t.Errorf("expected status CANCELLED, got %s", updated.Status)
	}

	ok, err := f.availability.IsAvailable(f.roomTypeID, f.dateRange)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected inventory to be released after cancellation")
	}
}

func TestService_Cancel_Rejected_WhenStillPendingPayment(t *testing.T) {
	f := newTestFixture(t)
	b, err := f.bookingSvc.CreateBooking("guest1", f.roomTypeID, f.dateRange, 2)
	if err != nil {
		t.Fatalf("fixture setup: unexpected error: %v", err)
	}
	// Never confirmed.

	_, err = f.refundSvc.Cancel(b.ID, time.Now())
	if err == nil {
		t.Fatal("expected cancelling a PENDING_PAYMENT booking to be rejected, got nil error")
	}
}

func TestService_Cancel_Rejected_WhenAlreadyCancelled_NoDoubleRefund(t *testing.T) {
	f := newTestFixture(t)
	b, err := f.bookingSvc.CreateBooking("guest1", f.roomTypeID, f.dateRange, 2)
	if err != nil {
		t.Fatalf("fixture setup: unexpected error: %v", err)
	}
	f.bookingSvc.ConfirmPayment(b.ID, "txn-1")

	if _, err := f.refundSvc.Cancel(b.ID, time.Now()); err != nil {
		t.Fatalf("fixture setup: unexpected error on first cancel: %v", err)
	}

	// The critical case: cancelling twice must not succeed a second
	// time - a second refund must never be issued.
	refund, err := f.refundSvc.Cancel(b.ID, time.Now())
	if err == nil {
		t.Fatal("expected cancelling an already-CANCELLED booking to be rejected, got nil error")
	}
	if refund != 0 {
		t.Errorf("expected no refund amount on a rejected cancellation, got %v", refund)
	}
}
