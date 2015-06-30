package schema

import "code.google.com/p/go-uuid/uuid"

// Mutation is an operation submitted to the application (most likely by a user)
// that alters the state of the datastore. Each mutation has a unique ID
// (which is a UUID v1 - therefore also has a rough timestamp encoded). Each
// mutation also contains the ID of the user who submitted the mutation.
type Mutation struct {
	ID     uuid.UUID
	UserID uuid.UUID
	Name   string
	Args   []byte
	Status string
}

// Arg is an argument for a mutation action.
type Arg interface{}

//
