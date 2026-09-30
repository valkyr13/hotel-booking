package refund

import (
	"time"

	"hotel-booking/booking"
	"hotel-booking/owner"
)

type Service struct {
	policy  Policy
	booking *booking.Service
}

func NewService(policy Policy, bookingService *booking.Service) *Service {
	return &Service{policy: policy, booking: bookingService}
}

func (s *Service) Cancel(bookingID string, cancelDate time.Time) (owner.Money, error) {
	b, err := s.booking.FindByID(bookingID)
	if err != nil {
		return 0, err
	}

	refundAmount := s.policy.CalculateRefund(b, cancelDate)

	if err := s.booking.CancelBooking(bookingID); err != nil {
		return 0, err
	}
	return refundAmount, nil
}
