package api

import (
	"net/http"

	"hotel-booking/availability"
	"hotel-booking/booking"
	"hotel-booking/owner"
	"hotel-booking/payment"
	"hotel-booking/refund"
	"hotel-booking/search"
)

type Server struct {
	ownerRepo    owner.OwnerRepository
	propertyRepo owner.PropertyRepository
	roomTypeRepo owner.RoomTypeRepository
	availability availability.Service
	bookingSvc   *booking.Service
	paymentSvc   *payment.Service
	refundSvc    *refund.Service
	searchSvc    *search.Service
}

// NewServer constructs a Server from its dependencies.
func NewServer(
	ownerRepo owner.OwnerRepository,
	propertyRepo owner.PropertyRepository,
	roomTypeRepo owner.RoomTypeRepository,
	avail availability.Service,
	bookingSvc *booking.Service,
	paymentSvc *payment.Service,
	refundSvc *refund.Service,
	searchSvc *search.Service,
) *Server {
	return &Server{
		ownerRepo:    ownerRepo,
		propertyRepo: propertyRepo,
		roomTypeRepo: roomTypeRepo,
		availability: avail,
		bookingSvc:   bookingSvc,
		paymentSvc:   paymentSvc,
		refundSvc:    refundSvc,
		searchSvc:    searchSvc,
	}
}

func (s *Server) Routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /owners", s.createOwner)
	mux.HandleFunc("POST /owners/{ownerID}/properties", s.addProperty)
	mux.HandleFunc("POST /properties/{propertyID}/room-types", s.addRoomType)

	mux.HandleFunc("GET /search", s.search)

	mux.HandleFunc("POST /bookings", s.createBooking)
	mux.HandleFunc("POST /bookings/{id}/pay", s.pay)
	mux.HandleFunc("POST /bookings/{id}/cancel", s.cancel)

	return mux
}
