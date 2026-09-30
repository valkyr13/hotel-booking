package owner

import (
	
	"fmt"
	"hotel-booking/utils"
)

type Owner struct {
	ID          string
	Name        string
	PropertyIDs []string
}


func NewOwner(name string) (*Owner, error) {
	if name == "" {
		return nil, fmt.Errorf("owner: name is required")
	}
	return &Owner{ID: utils.NewID(), Name: name}, nil
}

func (o *Owner) AddProperty(propertyID string) {
	o.PropertyIDs = append(o.PropertyIDs, propertyID)
}

func (o *Owner) IsSingleProperty() bool {
	return len(o.PropertyIDs) == 1
}
