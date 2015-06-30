package ident

import (
	"bitbucket.org/pkg/passwd"
	"fmt"
)

// A user identity
type Identity struct {
	Id       UUID
	Username string
	Password passwd.Passwd
	roles    map[string]*Role
	tokens   map[Token]bool
}

func NewIdentity(user string, password string) (id *Identity, err error) {
	id, err = newIdentity()
	if err != nil {
		return
	}
	id.Username = user
	id.Password, err = passwd.New([]byte(password))
	if err != nil {
		return
	}
	return
}

func newIdentity() (id *Identity, err error) {
	id = &Identity{}
	id.Id, err = NewUUID()
	if err != nil {
		return
	}
	id.roles = make(map[string]*Role)
	id.tokens = make(map[Token]bool)
	return
}

// Check that the identity is valid
func (id *Identity) Validate() (err error) {
	if id.Id == "" {
		return fmt.Errorf("Identity does not have a valid Id")
	}
	if id.Password == nil {
		return fmt.Errorf("Identity does not have a valid Password")
	}
	return
}

// Check if identity has a role
func (id *Identity) Is(role string) bool {
	_, ok := id.roles[role]
	return ok
}

// Assign a role
func (id *Identity) Grant(role string) (err error) {
	id.roles[role] = &Role{}
	return
}

// Unassign a role
func (id *Identity) Revoke(role string) (err error) {
	delete(id.roles, role)
	return
}
