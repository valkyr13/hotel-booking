package availability

import (
	"fmt"
	"sync"
	"time"

	"hotel-booking/owner"
)


type Service interface {
	Register(roomTypeID string, totalRooms int)
	IsAvailable(roomTypeID string, dr owner.DateRange) (bool, error)
	Reserve(roomTypeID string, dr owner.DateRange) (bool, error)
	Release(roomTypeID string, dr owner.DateRange) error
}

type nightlyAvailability struct {
	mu         sync.RWMutex
	totalRooms int
	booked     map[string]int 
}

type service struct {
	mu    sync.RWMutex 
	rooms map[string]*nightlyAvailability
}

func NewService() Service {
	return &service{rooms: make(map[string]*nightlyAvailability)}
}

func (s *service) Register(roomTypeID string, totalRooms int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rooms[roomTypeID] = &nightlyAvailability{
		totalRooms: totalRooms,
		booked:     make(map[string]int),
	}
}

func (s *service) find(roomTypeID string) (*nightlyAvailability, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	na, ok := s.rooms[roomTypeID]
	if !ok {
		return nil, fmt.Errorf("availability: room type %q was never registered", roomTypeID)
	}
	return na, nil
}

func (s *service) IsAvailable(roomTypeID string, dr owner.DateRange) (bool, error) {
	na, err := s.find(roomTypeID)
	if err != nil {
		return false, err
	}
	na.mu.RLock()
	defer na.mu.RUnlock()
	return na.hasRoomEveryNight(dr), nil
}

func (s *service) Reserve(roomTypeID string, dr owner.DateRange) (bool, error) {
	na, err := s.find(roomTypeID)
	if err != nil {
		return false, err
	}

	na.mu.Lock()
	defer na.mu.Unlock()

	if !na.hasRoomEveryNight(dr) {
		return false, nil
	}
	for _, night := range dr.Nights() {
		na.booked[dateKey(night)]++
	}
	return true, nil
}

func (s *service) Release(roomTypeID string, dr owner.DateRange) error {
	na, err := s.find(roomTypeID)
	if err != nil {
		return err
	}

	na.mu.Lock()
	defer na.mu.Unlock()

	for _, night := range dr.Nights() {
		key := dateKey(night)
		if na.booked[key] > 0 {
			na.booked[key]--
		}
		if na.booked[key] == 0 {
			delete(na.booked, key) 
		}
	}
	return nil
}

func (na *nightlyAvailability) hasRoomEveryNight(dr owner.DateRange) bool {
	for _, night := range dr.Nights() {
		remaining := na.totalRooms - na.booked[dateKey(night)]
		if remaining <= 0 {
			return false
		}
	}
	return true
}

func dateKey(t time.Time) string {
	return t.Format("2006-01-02")
}