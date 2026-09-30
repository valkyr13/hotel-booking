package booking

import (
	"errors"
	"sync"
)

var ErrNotFound = errors.New("booking: not found")

type Repository interface {
	Save(b *Booking) error
	FindByID(id string) (*Booking, error)
}

type inMemoryRepository struct {
	mu       sync.RWMutex
	bookings map[string]*Booking
}

// NewInMemoryRepository constructs an empty in-memory Repository.
func NewInMemoryRepository() Repository {
	return &inMemoryRepository{bookings: make(map[string]*Booking)}
}

func (r *inMemoryRepository) Save(b *Booking) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.bookings[b.ID] = b
	return nil
}

func (r *inMemoryRepository) FindByID(id string) (*Booking, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.bookings[id]
	if !ok {
		return nil, ErrNotFound
	}
	return b, nil
}
