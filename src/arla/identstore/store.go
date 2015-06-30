// Package identstore implements a user identity store to be used as a backend
// for user authentication.
package identstore

import (
	"arla/schema"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

// Store is the interface that backends adhere to
type Store interface {
	// Store a user
	Put(u *schema.User) error
	// Get a user for a given uuid or alias. returns nil if not found
	Get(alias string) *schema.User
}

// jsonfile implements Store using a flat json file
type jsonstore struct {
	filename string
	mu       sync.Mutex
	data     map[schema.UserID]*schema.User
}

// not safe to call without lock
func (s *jsonstore) load() error {
	// Parse existing file
	data := make(map[schema.UserID]*schema.User)
	f, err := os.OpenFile(s.filename, os.O_RDONLY, 0660)
	if err != nil {
		return err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	if err := dec.Decode(&data); err != nil {
		return err
	}
	s.data = data
	return nil
}

// not safe to call without lock
func (s *jsonstore) save() error {
	f, err := os.OpenFile(s.filename, os.O_WRONLY|os.O_TRUNC, 0660)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	if err := enc.Encode(s.data); err != nil {
		// if we fail here we could lose everything!! ... maybe
		// attempt to write the data elsewhere before crashing
		return err
	}
	return nil
}

// Register an ID/password combination
func (s *jsonstore) Put(u *schema.User) error {
	// Check ID is valid
	if !u.ID.Valid() {
		return fmt.Errorf("invalid user id")
	}
	// Check aliases don't conflict
	for _, alt := range u.Aliases {
		u2 := s.Get(alt)
		if u2 == nil {
			continue
		}
		if u2.ID != u.ID {
			return fmt.Errorf("another user already has alias %s", alt)
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[u.ID] = u
	return s.save()
}

// Find a user in the store. Alias can be a uuid or one of the aliases
// assigned to the user.
func (s *jsonstore) Get(alias string) *schema.User {
	u, ok := s.data[schema.UserID(alias)]
	if ok {
		return u
	}
	for _, u := range s.data {
		for _, alt := range u.Aliases {
			if strings.ToLower(alt) == strings.ToLower(alias) {
				return u
			}
		}
	}
	return nil
}

// Open sets up access to a Log for a given filename.
// If filename does not exist, it will be created.
// If a replay channel is returned, then each mutation read from disk
// will be sent to the chan. The chan will be closed once all mutations are read
// Writes will be buffered until all mutations are sent to the chan - so you MUST
// drain this chan before issusing writes!
func Open(filename string) (Store, error) {
	s := &jsonstore{
		filename: filename,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		f, err := os.OpenFile(s.filename, os.O_RDONLY|os.O_CREATE, 0660)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		s.data = make(map[schema.UserID]*schema.User)
		if err := s.save(); err != nil {
			return nil, err
		}
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}
