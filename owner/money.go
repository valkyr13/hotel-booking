package owner

import "fmt"

type Money int64

func NewMoney(rupees int64) (Money, error) {
	if rupees < 0 {
		return 0, fmt.Errorf("money: amount cannot be negative, got %d", rupees)
	}
	return Money(rupees), nil
}

func (m Money) Rupees() int64 { return int64(m) }

func (m Money) Add(other Money) Money { return m + other }

func (m Money) Subtract(other Money) Money {
	result := m - other
	if result < 0 {
		return 0
	}
	return result
}

func (m Money) Percentage(pct int) Money {
	return Money((int64(m)*int64(pct) + 50) / 100)
}

func (m Money) String() string {
	return fmt.Sprintf("Rs.%d", int64(m))
}
