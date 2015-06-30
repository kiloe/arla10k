package schema

import "code.google.com/p/go.crypto/bcrypt"

// UserID is the unique id type used for ident services
type UserID ID

// Valid returns true if the ID is safe to use
func (id UserID) Valid() bool {
	return id != ""
}

// User is a registered person with access to query arla
type User struct {
	ID      UserID
	Name    string
	Aliases []string
	Roles   []string
	Hash    []byte
}

// SetPassword accepts a plaintext password and store it internally as a hash
func (u *User) SetPassword(pw string) (err error) {
	u.Hash, err = bcrypt.GenerateFromPassword([]byte(pw), 10)
	return
}

// MatchPassword plaintext password pw with internal hash.
func (u *User) MatchPassword(pw string) bool {
	if len(u.Hash) == 0 {
		return false
	}
	return bcrypt.CompareHashAndPassword(u.Hash, []byte(pw)) == nil
}
