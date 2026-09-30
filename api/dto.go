package api

type createOwnerRequest struct {
	Name string `json:"name"`
}

type ownerResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type addPropertyRequest struct {
	Name       string   `json:"name"`
	City       string   `json:"city"`
	Locality   string   `json:"locality"`
	StarRating int      `json:"starRating"`
	Amenities  []string `json:"amenities"`
}

type propertyResponse struct {
	ID         string   `json:"id"`
	OwnerID    string   `json:"ownerId"`
	Name       string   `json:"name"`
	City       string   `json:"city"`
	Locality   string   `json:"locality"`
	StarRating int      `json:"starRating"`
	Amenities  []string `json:"amenities"`
}

type addRoomTypeRequest struct {
	Name         string   `json:"name"`
	MaxOccupancy int      `json:"maxOccupancy"`
	BasePrice    int64    `json:"basePrice"`
	Amenities    []string `json:"amenities"`
	RoomCount    int      `json:"roomCount"`
}

type roomTypeResponse struct {
	ID           string   `json:"id"`
	PropertyID   string   `json:"propertyId"`
	Name         string   `json:"name"`
	MaxOccupancy int      `json:"maxOccupancy"`
	BasePrice    int64    `json:"basePrice"`
	Amenities    []string `json:"amenities"`
	RoomCount    int      `json:"roomCount"`
}

type searchResultResponse struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	City       string   `json:"city"`
	Locality   string   `json:"locality"`
	StarRating int      `json:"starRating"`
	Amenities  []string `json:"amenities"`
}

type createBookingRequest struct {
	GuestID    string `json:"guestId"`
	RoomTypeID string `json:"roomTypeId"`
	CheckIn    string `json:"checkIn"` // YYYY-MM-DD
	CheckOut   string `json:"checkOut"`
	Guests     int    `json:"guests"`
}

type bookingResponse struct {
	ID            string `json:"id"`
	GuestID       string `json:"guestId"`
	RoomTypeID    string `json:"roomTypeId"`
	CheckIn       string `json:"checkIn"`
	CheckOut      string `json:"checkOut"`
	Guests        int    `json:"guests"`
	Amount        int64  `json:"amount"`
	Status        string `json:"status"`
	TransactionID string `json:"transactionId,omitempty"`
}

type payRequest struct {
	UserID string `json:"userId"`
}

type cancelResponse struct {
	Status       string `json:"status"`
	RefundAmount int64  `json:"refundAmount"`
}

type errorResponse struct {
	Error string `json:"error"`
}
