package payment

import "hotel-booking/owner"

type Method interface {
	Pay(userID, bookingID string, amount owner.Money) (success bool, transactionID string, err error)
}

type CardPayment struct {
	gateway Gateway
}

func NewCardPayment(gateway Gateway) Method {
	return &CardPayment{gateway: gateway}
}

func (c *CardPayment) Pay(userID, bookingID string, amount owner.Money) (bool, string, error) {
	approved, transactionID, err := c.gateway.Charge(amount)
	if err != nil {
		return false, "", err
	}
	if !approved {
		return false, "", nil
	}
	return true, transactionID, nil
}
