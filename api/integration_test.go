package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"hotel-booking/availability"
	"hotel-booking/booking"
	"hotel-booking/owner"
	"hotel-booking/payment"
	"hotel-booking/refund"
	"hotel-booking/search"
)

func newTestServer() *httptest.Server {
	ownerRepo := owner.NewInMemoryOwnerRepository()
	propertyRepo := owner.NewInMemoryPropertyRepository()
	roomTypeRepo := owner.NewInMemoryRoomTypeRepository()
	bookingRepo := booking.NewInMemoryRepository()

	availabilitySvc := availability.NewService()
	bookingSvc := booking.NewService(roomTypeRepo, availabilitySvc, bookingRepo)
	paymentSvc := payment.NewService(payment.NewCardPayment(payment.NewAlwaysApproveGateway()), bookingSvc)
	refundSvc := refund.NewService(refund.NewTieredRefundPolicy(), bookingSvc)
	searchSvc := search.NewService(propertyRepo)

	server := NewServer(ownerRepo, propertyRepo, roomTypeRepo, availabilitySvc, bookingSvc, paymentSvc, refundSvc, searchSvc)
	return httptest.NewServer(server.Routes())
}

func postJSON(t *testing.T, url string, body any, out any) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
	}
	return resp
}

func TestEndToEnd_OnboardSearchBookPayCancel(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	// 1. Create owner.
	var ownerResp ownerResponse
	resp := postJSON(t, srv.URL+"/owners", createOwnerRequest{Name: "Vee's Hotels"}, &ownerResp)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating owner, got %d", resp.StatusCode)
	}

	// 2. Add property.
	var propResp propertyResponse
	resp = postJSON(t, srv.URL+"/owners/"+ownerResp.ID+"/properties", addPropertyRequest{
		Name: "Taj Residency", City: "Bengaluru", Locality: "MG Road", StarRating: 4, Amenities: []string{"POOL"},
	}, &propResp)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 adding property, got %d", resp.StatusCode)
	}

	// 3. Add room type - this also registers it with availability.
	var rtResp roomTypeResponse
	resp = postJSON(t, srv.URL+"/properties/"+propResp.ID+"/room-types", addRoomTypeRequest{
		Name: "Deluxe", MaxOccupancy: 2, BasePrice: 3000, Amenities: []string{"AC"}, RoomCount: 2,
	}, &rtResp)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 adding room type, got %d", resp.StatusCode)
	}

	// 4. Search should find it.
	searchURL := srv.URL + "/search?city=Bengaluru&checkIn=2027-03-05&checkOut=2027-03-08&guests=2"
	searchResp, err := http.Get(searchURL)
	if err != nil {
		t.Fatalf("search request failed: %v", err)
	}
	var results []searchResultResponse
	if err := json.NewDecoder(searchResp.Body).Decode(&results); err != nil {
		t.Fatalf("failed to decode search response: %v", err)
	}
	if len(results) != 1 || results[0].ID != propResp.ID {
		t.Fatalf("expected search to find the newly added property, got %+v", results)
	}

	// 5. Create booking.
	var bookingResp bookingResponse
	resp = postJSON(t, srv.URL+"/bookings", createBookingRequest{
		GuestID: "guest1", RoomTypeID: rtResp.ID, CheckIn: "2027-03-05", CheckOut: "2027-03-08", Guests: 2,
	}, &bookingResp)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating booking, got %d", resp.StatusCode)
	}
	if bookingResp.Status != "PENDING_PAYMENT" {
		t.Errorf("expected status PENDING_PAYMENT, got %s", bookingResp.Status)
	}
	if bookingResp.Amount != 9000 { // 3000/night x 3 nights
		t.Errorf("expected amount 9000, got %d", bookingResp.Amount)
	}

	// 6. Pay.
	var paidResp bookingResponse
	resp = postJSON(t, srv.URL+"/bookings/"+bookingResp.ID+"/pay", payRequest{UserID: "guest1"}, &paidResp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 paying, got %d", resp.StatusCode)
	}
	if paidResp.Status != "CONFIRMED" {
		t.Errorf("expected status CONFIRMED, got %s", paidResp.Status)
	}
	if paidResp.TransactionID == "" {
		t.Error("expected a transaction ID after successful payment")
	}

	// 7. Cancel - check-in is many months out, so this lands in the
	// top refund tier (90%).
	var cancelResp cancelResponse
	resp = postJSON(t, srv.URL+"/bookings/"+bookingResp.ID+"/cancel", nil, &cancelResp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 cancelling, got %d", resp.StatusCode)
	}
	if cancelResp.Status != "CANCELLED" {
		t.Errorf("expected status CANCELLED, got %s", cancelResp.Status)
	}
	if cancelResp.RefundAmount != 8100 { // 90% of 9000
		t.Errorf("expected refund 8100, got %d", cancelResp.RefundAmount)
	}
}

func TestCreateBooking_UnknownRoomType_Returns404(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	resp := postJSON(t, srv.URL+"/bookings", createBookingRequest{
		GuestID: "guest1", RoomTypeID: "does-not-exist", CheckIn: "2027-03-05", CheckOut: "2027-03-08", Guests: 2,
	}, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for unknown room type, got %d", resp.StatusCode)
	}
}

func TestPay_AlreadyConfirmedBooking_ReturnsError(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	var ownerResp ownerResponse
	postJSON(t, srv.URL+"/owners", createOwnerRequest{Name: "Vee's Hotels"}, &ownerResp)
	var propResp propertyResponse
	postJSON(t, srv.URL+"/owners/"+ownerResp.ID+"/properties", addPropertyRequest{
		Name: "Taj Residency", City: "Bengaluru", StarRating: 4,
	}, &propResp)
	var rtResp roomTypeResponse
	postJSON(t, srv.URL+"/properties/"+propResp.ID+"/room-types", addRoomTypeRequest{
		Name: "Deluxe", MaxOccupancy: 2, BasePrice: 3000, RoomCount: 1,
	}, &rtResp)
	var bookingResp bookingResponse
	postJSON(t, srv.URL+"/bookings", createBookingRequest{
		GuestID: "guest1", RoomTypeID: rtResp.ID, CheckIn: "2027-03-05", CheckOut: "2027-03-08", Guests: 2,
	}, &bookingResp)
	postJSON(t, srv.URL+"/bookings/"+bookingResp.ID+"/pay", payRequest{UserID: "guest1"}, nil)

	// Second pay attempt on an already-CONFIRMED booking must fail.
	resp := postJSON(t, srv.URL+"/bookings/"+bookingResp.ID+"/pay", payRequest{UserID: "guest1"}, nil)
	if resp.StatusCode == http.StatusOK {
		t.Error("expected paying an already-confirmed booking to fail")
	}
}
