package main

import (
	"log"
	"net/http"

	"hotel-booking/api"
	"hotel-booking/availability"
	"hotel-booking/booking"
	"hotel-booking/owner"
	"hotel-booking/payment"
	"hotel-booking/refund"
	"hotel-booking/search"
)

func main() {
	// --- Repositories (in-memory; each package owns its own) ---
	ownerRepo := owner.NewInMemoryOwnerRepository()
	propertyRepo := owner.NewInMemoryPropertyRepository()
	roomTypeRepo := owner.NewInMemoryRoomTypeRepository()
	bookingRepo := booking.NewInMemoryRepository()

	// --- Core services ---
	availabilitySvc := availability.NewService()
	bookingSvc := booking.NewService(roomTypeRepo, availabilitySvc, bookingRepo)

	cardPayment := payment.NewCardPayment(payment.NewAlwaysApproveGateway())
	paymentSvc := payment.NewService(cardPayment, bookingSvc)

	refundPolicy := refund.NewTieredRefundPolicy()
	refundSvc := refund.NewService(refundPolicy, bookingSvc)

	searchSvc := search.NewService(propertyRepo)

	// --- HTTP server ---
	server := api.NewServer(ownerRepo, propertyRepo, roomTypeRepo, availabilitySvc, bookingSvc, paymentSvc, refundSvc, searchSvc)

	addr := ":8080"
	log.Printf("hotel-booking listening on %s", addr)
	if err := http.ListenAndServe(addr, server.Routes()); err != nil {
		log.Fatal(err)
	}
}
