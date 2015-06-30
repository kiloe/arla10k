package ident

import "testing"

func TestMemStorePut(t *testing.T) {
	ms := NewMemoryStore()
	u1, err := newIdentity()
	if err != nil {
		t.Error(err)
		return
	}
	u1.Username = "Jeff"
	ms.Put(u1)
	u2, err := ms.FindById(u1.Id)
	if err != nil {
		t.Error(err)
		return
	}
	if u2.Id != u1.Id {
		t.Error("Expected Put Id %v to matched returned Id. Got: %v", u1.Id, u2.Id)
		return
	}
}

func TestMemStoreFindByUsername(t *testing.T) {
	ms := NewMemoryStore()
	u1, err := newIdentity()
	if err != nil {
		t.Error(err)
		return
	}
	u1.Username = "Jeff"
	ms.Put(u1)
	u2, err := ms.FindByUsername("jeff") // case insentitve
	if err != nil {
		t.Error(err)
		return
	}
	if u2.Id != u1.Id {
		t.Error("Expected Put Id %v to matched returned Id. Got: %v", u1.Id, u2.Id)
		return
	}
}
