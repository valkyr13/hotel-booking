package owner

import (
	"fmt"
	"time"
)

type DateRange struct {
	CheckIn  time.Time
	CheckOut time.Time
}

func NewDateRange(checkIn, checkOut time.Time) (DateRange, error) {
	checkIn = truncateToDay(checkIn)
	checkOut = truncateToDay(checkOut)
	if !checkOut.After(checkIn) {
		return DateRange{}, fmt.Errorf("daterange: check-out %s must be after check-in %s",
			checkOut.Format("2006-01-02"), checkIn.Format("2006-01-02"))
	}
	return DateRange{CheckIn: checkIn, CheckOut: checkOut}, nil
}

func (d DateRange) Nights() []time.Time {
	var nights []time.Time
	for t := d.CheckIn; t.Before(d.CheckOut); t = t.AddDate(0, 0, 1) {
		nights = append(nights, t)
	}
	return nights
}

func (d DateRange) Overlaps(other DateRange) bool {
	return d.CheckIn.Before(other.CheckOut) && other.CheckIn.Before(d.CheckOut)
}

func (d DateRange) DaysUntil(reference time.Time) int {
	reference = truncateToDay(reference)
	return int(d.CheckIn.Sub(reference).Hours() / 24)
}

func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
