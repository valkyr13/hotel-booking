package owner

import "testing"

func TestOwnerRepository_SaveAndFindByID(t *testing.T) {
	repo := NewInMemoryOwnerRepository()
	o, _ := NewOwner("Vee's Hotels")

	if err := repo.Save(o); err != nil {
		t.Fatalf("unexpected error on save: %v", err)
	}
	got, err := repo.FindByID(o.ID)
	if err != nil {
		t.Fatalf("unexpected error on find: %v", err)
	}
	if got.Name != "Vee's Hotels" {
		t.Errorf("expected name %q, got %q", "Vee's Hotels", got.Name)
	}
}

func TestOwnerRepository_FindByID_NotFound(t *testing.T) {
	repo := NewInMemoryOwnerRepository()
	_, err := repo.FindByID("does-not-exist")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestPropertyRepository_FindAll(t *testing.T) {
	repo := NewInMemoryPropertyRepository()
	p1, _ := NewProperty("o1", "Taj Residency", "Bengaluru", "MG Road", 4, nil)
	p2, _ := NewProperty("o1", "Budget Inn", "Mumbai", "Andheri", 2, nil)
	repo.Save(p1)
	repo.Save(p2)

	all, err := repo.FindAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 properties, got %d", len(all))
	}
}

func TestRoomTypeRepository_FindByPropertyID(t *testing.T) {
	repo := NewInMemoryRoomTypeRepository()
	rt1, _ := NewRoomType("prop1", "Deluxe", 2, Money(3000), nil, 3)
	rt2, _ := NewRoomType("prop1", "Suite", 4, Money(8000), nil, 1)
	rt3, _ := NewRoomType("prop2", "Single", 1, Money(1500), nil, 5)
	repo.Save(rt1)
	repo.Save(rt2)
	repo.Save(rt3)

	forProp1, err := repo.FindByPropertyID("prop1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(forProp1) != 2 {
		t.Fatalf("expected 2 room types for prop1, got %d", len(forProp1))
	}

	forProp2, err := repo.FindByPropertyID("prop2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(forProp2) != 1 {
		t.Fatalf("expected 1 room type for prop2, got %d", len(forProp2))
	}
}