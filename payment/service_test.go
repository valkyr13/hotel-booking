package payment

import (
	"testing"
	"time"

	"hotel-booking/availability"
	"hotel-booking/booking"
	"hotel-booking/owner"
)

func TestCardPayment_Success(t *testing.T) {
	c := NewCardPayment(NewAlwaysApproveGateway())
	success, txID, err := c.Pay("user1", "booking1", owner.Money(3000))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !success {
		t.Error("expected success with an always-approve gateway")
	}
	if txID == "" {
		t.Error("expected a non-empty transaction ID on success")
	}
}

func TestCardPayment_Declined(t *testing.T) {
	c := NewCardPayment(NewAlwaysDeclineGateway())
	success, txID, err := c.Pay("user1", "booking1", owner.Money(3000))
	if err != nil {
		t.Fatalf("expected no Go error on a legitimate decline, got %v", err)
	}
	if success {
		t.Error("expected failure with an always-decline gateway")
	}
	if txID != "" {
		t.Error("expected no transaction ID on a declined charge")
	}
}

func TestCardPayment_GatewayUnreachable(t *testing.T) {
	c := NewCardPayment(NewUnreachableGateway())
	success, _, err := c.Pay("user1", "booking1", owner.Money(3000))
	if err == nil {
		t.Fatal("expected an error when the gateway is unreachable, got nil")
	}
	if success {
		t.Error("expected failure when the gateway is unreachable")
	}
}

type testFixture struct {
	bookingSvc   *booking.Service
	availability availability.Service
	roomTypeID   string
	dateRange    owner.DateRange
}

func newTestFixture(t *testing.T) *testFixture {
	t.Helper()

	roomTypeRepo := owner.NewInMemoryRoomTypeRepository()
	rt, _ := owner.NewRoomType("prop1", "Deluxe", 2, owner.Money(3000), nil, 1) // 1 room, so release is observable
	roomTypeRepo.Save(rt)

	avail := availability.NewService()
	avail.Register(rt.ID, 1)

	dr, _ := owner.NewDateRange(
		time.Date(2027, 3, 5, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 3, 8, 0, 0, 0, 0, time.UTC),
	)

	return &testFixture{
		bookingSvc:   booking.NewService(roomTypeRepo, avail, booking.NewInMemoryRepository()),
		availability: avail,
		roomTypeID:   rt.ID,
		dateRange:    dr,
	}
}

func TestPaymentService_Pay_Success_ConfirmsBooking(t *testing.T) {
	f := newTestFixture(t)
	b, err := f.bookingSvc.CreateBooking("guest1", f.roomTypeID, f.dateRange, 2)
	if err != nil {
		t.Fatalf("fixture setup: unexpected error: %v", err)
	}

	paymentSvc := NewService(NewCardPayment(NewAlwaysApproveGateway()), f.bookingSvc)
	if err := paymentSvc.Pay("guest1", b); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	ok, err := f.availability.IsAvailable(f.roomTypeID, f.dateRange)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected room to remain reserved after successful payment, not released")
	}
}

func TestPaymentService_Pay_Declined_FailsBookingAndReleasesInventory(t *testing.T) {
	f := newTestFixture(t)
	b, err := f.bookingSvc.CreateBooking("guest1", f.roomTypeID, f.dateRange, 2)
	if err != nil {
		t.Fatalf("fixture setup: unexpected error: %v", err)
	}

	paymentSvc := NewService(NewCardPayment(NewAlwaysDeclineGateway()), f.bookingSvc)
	if err := paymentSvc.Pay("guest1", b); err == nil {
		t.Fatal("expected an error to be returned on decline")
	}

	ok, err := f.availability.IsAvailable(f.roomTypeID, f.dateRange)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected room to be released back to available after a declined payment")
	}
}

func TestPaymentService_Pay_GatewayError_FailsBookingAndReleasesInventory(t *testing.T) {
	f := newTestFixture(t)
	b, err := f.bookingSvc.CreateBooking("guest1", f.roomTypeID, f.dateRange, 2)
	if err != nil {
		t.Fatalf("fixture setup: unexpected error: %v", err)
	}

	paymentSvc := NewService(NewCardPayment(NewUnreachableGateway()), f.bookingSvc)
	if err := paymentSvc.Pay("guest1", b); err == nil {
		t.Fatal("expected an error to be returned when the gateway is unreachable")
	}

	ok, err := f.availability.IsAvailable(f.roomTypeID, f.dateRange)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected room to be released back to available after a gateway error, same as a decline")
	}
}
