package owner

import (
	"errors"
	"sync"
)


var ErrNotFound = errors.New("owner: not found")

// --------------------------------------------------------- Owner repository ------------------------------------------------------------------------------

type OwnerRepository interface {
	Save(o *Owner) error
	FindByID(id string) (*Owner, error)
}

type inMemoryOwnerRepository struct {
	mu     sync.RWMutex
	owners map[string]*Owner
}

func NewInMemoryOwnerRepository() OwnerRepository {
	return &inMemoryOwnerRepository{owners: make(map[string]*Owner)}
}

func (r *inMemoryOwnerRepository) Save(o *Owner) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.owners[o.ID] = o
	return nil
}

func (r *inMemoryOwnerRepository) FindByID(id string) (*Owner, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o, ok := r.owners[id]
	if !ok {
		return nil, ErrNotFound
	}
	return o, nil
}

// -------------------------------------------------------------------------Property repository -------------------------------------------------------------

type PropertyRepository interface {
	Save(p *Property) error
	FindByID(id string) (*Property, error)
	FindAll() ([]*Property, error)
}

type inMemoryPropertyRepository struct {
	mu         sync.RWMutex
	properties map[string]*Property
}


func NewInMemoryPropertyRepository() PropertyRepository {
	return &inMemoryPropertyRepository{properties: make(map[string]*Property)}
}

func (r *inMemoryPropertyRepository) Save(p *Property) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.properties[p.ID] = p
	return nil
}

func (r *inMemoryPropertyRepository) FindByID(id string) (*Property, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.properties[id]
	if !ok {
		return nil, ErrNotFound
	}
	return p, nil
}

func (r *inMemoryPropertyRepository) FindAll() ([]*Property, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	all := make([]*Property, 0, len(r.properties))
	for _, p := range r.properties {
		all = append(all, p)
	}
	return all, nil
}

// -----------------------------------------------------------------RoomType repository ------------------------------------------------------------------------

type RoomTypeRepository interface {
	Save(rt *RoomType) error
	FindByID(id string) (*RoomType, error)
	FindByPropertyID(propertyID string) ([]*RoomType, error)
}

type inMemoryRoomTypeRepository struct {
	mu        sync.RWMutex
	roomTypes map[string]*RoomType
}


func NewInMemoryRoomTypeRepository() RoomTypeRepository {
	return &inMemoryRoomTypeRepository{roomTypes: make(map[string]*RoomType)}
}

func (r *inMemoryRoomTypeRepository) Save(rt *RoomType) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.roomTypes[rt.ID] = rt
	return nil
}

func (r *inMemoryRoomTypeRepository) FindByID(id string) (*RoomType, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rt, ok := r.roomTypes[id]
	if !ok {
		return nil, ErrNotFound
	}
	return rt, nil
}

func (r *inMemoryRoomTypeRepository) FindByPropertyID(propertyID string) ([]*RoomType, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*RoomType
	for _, rt := range r.roomTypes {
		if rt.PropertyID == propertyID {
			result = append(result, rt)
		}
	}
	return result, nil
}