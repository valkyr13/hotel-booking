package search

import "hotel-booking/owner"

// interface. See the Unit 6 design.
type Service struct {
	properties owner.PropertyRepository
}

func NewService(properties owner.PropertyRepository) *Service {
	return &Service{properties: properties}
}

func (s *Service) Search(filters ...Filter) ([]*owner.Property, error) {
	all, err := s.properties.FindAll()
	if err != nil {
		return nil, err
	}
	return applyFilters(all, filters), nil
}

func applyFilters(properties []*owner.Property, filters []Filter) []*owner.Property {
	result := properties
	for _, f := range filters {
		var survivors []*owner.Property
		for _, p := range result {
			if f.Matches(p) {
				survivors = append(survivors, p)
			}
		}
		result = survivors
	}
	return result
}
