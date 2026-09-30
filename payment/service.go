package payment

import (
	"fmt"

	"hotel-booking/booking"
)

type Service struct {
	method  Method
	booking *booking.Service
}

func NewService(method Method, bookingService *booking.Service) *Service {
	return &Service{method: method, booking: bookingService}
}

func (s *Service) Pay(userID string, b *booking.Booking) error {
	success, transactionID, err := s.method.Pay(userID, b.ID, b.Amount)
	if err != nil || !success {
		if failErr := s.booking.FailPayment(b.ID); failErr != nil {
			return fmt.Errorf("payment: charge failed (%v) and booking transition also failed: %w", err, failErr)
		}
		if err != nil {
			return fmt.Errorf("payment: gateway error: %w", err)
		}
		return fmt.Errorf("payment: declined")
	}

	if err := s.booking.ConfirmPayment(b.ID, transactionID); err != nil {
		return fmt.Errorf("payment: charge succeeded but booking confirmation failed: %w", err)
	}
	return nil
}
