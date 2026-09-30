package api

import (
	"encoding/json"
	"net/http"
	"time"

	"hotel-booking/booking"
	"hotel-booking/owner"
)

func (s *Server) createBooking(w http.ResponseWriter, r *http.Request) {
	var req createBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, err)
		return
	}

	checkIn, err := parseDate(req.CheckIn)
	if err != nil {
		writeError(w, err)
		return
	}
	checkOut, err := parseDate(req.CheckOut)
	if err != nil {
		writeError(w, err)
		return
	}
	dr, err := owner.NewDateRange(checkIn, checkOut)
	if err != nil {
		writeError(w, err)
		return
	}

	b, err := s.bookingSvc.CreateBooking(req.GuestID, req.RoomTypeID, dr, req.Guests)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toBookingResponse(b))
}

func (s *Server) pay(w http.ResponseWriter, r *http.Request) {
	bookingID := r.PathValue("id")

	b, err := s.bookingSvc.FindByID(bookingID)
	if err != nil {
		writeError(w, err)
		return
	}

	var req payRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, err)
		return
	}

	if err := s.paymentSvc.Pay(req.UserID, b); err != nil {
		writeError(w, err)
		return
	}

	updated, err := s.bookingSvc.FindByID(bookingID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toBookingResponse(updated))
}

func (s *Server) cancel(w http.ResponseWriter, r *http.Request) {
	bookingID := r.PathValue("id")

	refundAmount, err := s.refundSvc.Cancel(bookingID, time.Now())
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, cancelResponse{Status: "CANCELLED", RefundAmount: refundAmount.Rupees()})
}

func toBookingResponse(b *booking.Booking) bookingResponse {
	return bookingResponse{
		ID: b.ID, GuestID: b.GuestID, RoomTypeID: b.RoomTypeID,
		CheckIn: b.DateRange.CheckIn.Format("2006-01-02"), CheckOut: b.DateRange.CheckOut.Format("2006-01-02"),
		Guests: b.GuestCount, Amount: b.Amount.Rupees(), Status: string(b.Status), TransactionID: b.TransactionID,
	}
}
