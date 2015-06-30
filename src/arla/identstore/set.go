package ident

import (
	"fmt"
	"sync"
	"time"
)

// A collection of identities
type Set struct {
	tokens        map[Token]*Identity
	tokenDuration string
	mu            sync.Mutex
	Storer
}

// Add an identity to the set
func (s *Set) Add(id *Identity) (err error) {
	if id == nil {
		return fmt.Errorf("Cannot add nil identity")
	}
	if err = id.Validate(); err != nil {
		return err
	}
	if _, err := s.FindByUsername(id.Username); err == nil {
		return fmt.Errorf("User with name %v already exists", id.Username)
	}
	if _, err := s.FindById(id.Id); err == nil {
		return fmt.Errorf("User with id %v already exists", id.Id)
	}
	if err = s.Put(id); err != nil {
		return err
	}
	return
}

// Remove an identity from the set
func (s *Set) Remove(id *Identity) (err error) {
	if id == nil {
		return fmt.Errorf("Cannot remove nil identity")
	}
	s.logout(id, "")
	return s.Storer.Delete(id)
}

// Update an existing entity in the set
func (s *Set) Update(id *Identity) (err error) {
	if id == nil {
		return fmt.Errorf("Cannot update nil identity")
	}
	if err = id.Validate(); err != nil {
		return err
	}
	return s.Storer.Put(id)
}

// Fetch identity and token for given username and password.
// Token can be used to lookup identity for <tokenDuration>
func (s *Set) Login(username, password string) (id *Identity, tok Token, err error) {
	if id, err = s.FindByUsername(username); err != nil {
		return
	}
	if !id.Password.Test([]byte(password)) {
		return nil, tok, fmt.Errorf("Password did not match")
	}
	if tok, err = NewToken(); err != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id.tokens[tok] = true
	s.tokens[tok] = id
	d, err := time.ParseDuration(s.tokenDuration)
	if err != nil {
		return
	}
	time.AfterFunc(d, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.tokens, tok)
		delete(id.tokens, tok)
	})
	return
}

// Revoke a login token. If `all` is `true` then revoke all tokens.
func (s *Set) Logout(tok Token, all bool) (err error) {
	id, err := s.FindByToken(tok)
	if err != nil {
		return
	}
	if all {
		tok = ""
	}
	s.logout(id, tok)
	return
}

// Remove auth-tokens
func (s *Set) logout(id *Identity, tok Token) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for t, _ := range id.tokens {
		if tok != "" && tok != t {
			continue
		}
		delete(s.tokens, t)
		delete(id.tokens, t)
	}
	return
}

func (s *Set) FindByToken(tok Token) (id *Identity, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.tokens[tok]
	if !ok {
		return nil, fmt.Errorf("No identity found for authentication token %s", tok)
	}
	return
}

// Create a new identity set
func NewSet(store Storer) (s *Set, err error) {
	s = &Set{}
	s.Storer = store
	s.tokenDuration = "300h"
	s.tokens = make(map[Token]*Identity)
	return
}
