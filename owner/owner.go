package owner

import "fmt"

type Owner struct {
	ID          string
	Name        string
	PropertyIDs []string
}


func NewOwner(name string) (*Owner, error) {
	if name == "" {
		return nil, fmt.Errorf("owner: name is required")
	}
	return &Owner{ID: newID(), Name: name}, nil
}

func (o *Owner) AddProperty(propertyID string) {
	o.PropertyIDs = append(o.PropertyIDs, propertyID)
}

func (o *Owner) IsSingleProperty() bool {
	return len(o.PropertyIDs) == 1
}
