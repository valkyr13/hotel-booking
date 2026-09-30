package booking

import (
	"fmt"

	"hotel-booking/availability"
	"hotel-booking/owner"
)

type Service struct {
	roomTypes    owner.RoomTypeRepository
	availability availability.Service
	bookings     Repository
}

func NewService(roomTypes owner.RoomTypeRepository, avail availability.Service, bookings Repository) *Service {
	return &Service{roomTypes: roomTypes, availability: avail, bookings: bookings}
}

func (s *Service) CreateBooking(guestID, roomTypeID string, dr owner.DateRange, guestCount int) (*Booking, error) {
	rt, err := s.roomTypes.FindByID(roomTypeID)
	if err != nil {
		return nil, fmt.Errorf("booking: room type lookup failed: %w", err)
	}
	if !rt.FitsGuests(guestCount) {
		return nil, fmt.Errorf("booking: room type %s does not fit %d guests (max %d)", roomTypeID, guestCount, rt.MaxOccupancy)
	}

	ok, err := s.availability.Reserve(roomTypeID, dr)
	if err != nil {
		return nil, fmt.Errorf("booking: reserve failed: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("booking: no rooms available for room type %s in the requested date range", roomTypeID)
	}

	amount := rt.BasePrice.Multiply(len(dr.Nights()))
	b := newBooking(guestID, roomTypeID, dr, guestCount, amount)

	if err := s.bookings.Save(b); err != nil {

		_ = s.availability.Release(roomTypeID, dr)
		return nil, fmt.Errorf("booking: failed to save booking: %w", err)
	}
	return b, nil
}

func (s *Service) ConfirmPayment(bookingID, transactionID string) error {
	b, err := s.bookings.FindByID(bookingID)
	if err != nil {
		return err
	}
	if err := b.MarkConfirmed(); err != nil {
		return err
	}
	b.TransactionID = transactionID
	return s.bookings.Save(b)
}

func (s *Service) FailPayment(bookingID string) error {
	b, err := s.bookings.FindByID(bookingID)
	if err != nil {
		return err
	}
	if err := b.MarkFailed(); err != nil {
		return err
	}
	if err := s.availability.Release(b.RoomTypeID, b.DateRange); err != nil {
		return fmt.Errorf("booking: failed to release inventory after payment failure: %w", err)
	}
	return s.bookings.Save(b)
}

func (s *Service) CancelBooking(bookingID string) error {
	b, err := s.bookings.FindByID(bookingID)
	if err != nil {
		return err
	}
	if err := b.Cancel(); err != nil {
		return err
	}
	if err := s.availability.Release(b.RoomTypeID, b.DateRange); err != nil {
		return fmt.Errorf("booking: failed to release inventory after cancellation: %w", err)
	}
	return s.bookings.Save(b)
}

func (s *Service) FindByID(bookingID string) (*Booking, error) {
	return s.bookings.FindByID(bookingID)
}
