package payment

import (
	"crypto/rand"
	"encoding/hex"
	"errors"

	"hotel-booking/owner"
)

var errUnreachable = errors.New("payment: gateway unreachable")


type Gateway interface {
	Charge(amount owner.Money) (approved bool, transactionID string, err error)
}

type MockGateway struct {
	ChargeFunc func(amount owner.Money) (bool, string, error)
}

func (g *MockGateway) Charge(amount owner.Money) (bool, string, error) {
	if g.ChargeFunc != nil {
		return g.ChargeFunc(amount)
	}
	return true, newTransactionID(), nil 
}

func NewAlwaysApproveGateway() Gateway {
	return &MockGateway{ChargeFunc: func(owner.Money) (bool, string, error) {
		return true, newTransactionID(), nil
	}}
}


func NewAlwaysDeclineGateway() Gateway {
	return &MockGateway{ChargeFunc: func(owner.Money) (bool, string, error) {
		return false, "", nil
	}}
}


func NewUnreachableGateway() Gateway {
	return &MockGateway{ChargeFunc: func(owner.Money) (bool, string, error) {
		return false, "", errUnreachable
	}}
}

func newTransactionID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "txn_" + hex.EncodeToString(b)
}