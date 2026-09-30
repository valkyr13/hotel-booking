package refund

import (
	"time"

	"hotel-booking/booking"
	"hotel-booking/owner"
)

type Policy interface {
	CalculateRefund(b *booking.Booking, cancelDate time.Time) owner.Money
}

type TieredRefundPolicy struct{}

func NewTieredRefundPolicy() Policy {
	return TieredRefundPolicy{}
}

func (TieredRefundPolicy) CalculateRefund(b *booking.Booking, cancelDate time.Time) owner.Money {
	days := b.DateRange.DaysUntil(cancelDate)
	switch {
	case days >= 10:
		return b.Amount.Percentage(90)
	case days >= 3:
		return b.Amount.Percentage(80)
	case days >= 1:
		return b.Amount.Percentage(50)
	default:
		return b.Amount.Percentage(30)
	}
}
