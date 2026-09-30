package booking

import (
	"fmt"
	"hotel-booking/owner"
	"hotel-booking/utils"
	"time"
)

type Status string

const (
	StatusPendingPayment Status = "PENDING_PAYMENT"
	StatusConfirmed      Status = "CONFIRMED"
	StatusFailed         Status = "FAILED"
	StatusCancelled      Status = "CANCELLED"
)

type Booking struct {
	ID            string
	GuestID       string
	RoomTypeID    string
	DateRange     owner.DateRange
	GuestCount    int
	Amount        owner.Money
	Status        Status
	TransactionID string
	CreatedAt     time.Time
}

func newBooking(guestID, roomTypeID string, dr owner.DateRange, guestCount int, amount owner.Money) *Booking {
	return &Booking{
		ID:         utils.NewID(),
		GuestID:    guestID,
		RoomTypeID: roomTypeID,
		DateRange:  dr,
		GuestCount: guestCount,
		Amount:     amount,
		Status:     StatusPendingPayment,
		CreatedAt:  time.Now(),
	}
}

func (b *Booking) MarkConfirmed() error {
	if b.Status != StatusPendingPayment {
		return fmt.Errorf("booking: cannot confirm booking %s from status %s", b.ID, b.Status)
	}
	b.Status = StatusConfirmed
	return nil
}

func (b *Booking) MarkFailed() error {
	if b.Status != StatusPendingPayment {
		return fmt.Errorf("booking: cannot fail booking %s from status %s", b.ID, b.Status)
	}
	b.Status = StatusFailed
	return nil
}

func (b *Booking) Cancel() error {
	if b.Status != StatusConfirmed {
		return fmt.Errorf("booking: cannot cancel booking %s from status %s", b.ID, b.Status)
	}
	b.Status = StatusCancelled
	return nil
}
