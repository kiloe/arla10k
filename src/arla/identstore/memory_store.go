package ident

import (
	"fmt"
	"strings"
	"sync"
)

// A fake store that does nothing
type MemoryStore struct {
	ids   map[UUID]*Identity
	users map[string]*Identity
	mu    sync.Mutex
}

func NewMemoryStore() *MemoryStore {
	ms := &MemoryStore{}
	ms.load(nil)
	return ms
}

func (ms *MemoryStore) load(idents map[UUID]*Identity) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	if idents == nil {
		ms.ids = make(map[UUID]*Identity)
	} else {
		ms.ids = idents
	}
	ms.users = make(map[string]*Identity)
	for _, id := range ms.ids {
		ms.users[id.Username] = id
	}
}

func (ms *MemoryStore) Put(id *Identity) (err error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.ids[id.Id] = id
	ms.users[strings.ToLower(id.Username)] = id
	return
}

func (ms *MemoryStore) Delete(id *Identity) (err error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	delete(ms.ids, id.Id)
	delete(ms.users, id.Username)
	return
}

// Lookup an ident by uuid
func (ms *MemoryStore) FindById(uuid UUID) (id *Identity, err error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	id, ok := ms.ids[uuid]
	if !ok {
		return nil, fmt.Errorf("No identity found for id %s", uuid)
	}
	return
}

// Lookup an ident by username
func (ms *MemoryStore) FindByUsername(username string) (id *Identity, err error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	id, ok := ms.users[strings.ToLower(username)]
	if !ok {
		return nil, fmt.Errorf("No identity found for username %s", username)
	}
	return
}
