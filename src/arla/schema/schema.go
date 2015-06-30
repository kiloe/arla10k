package schema

import (
	"io"
	"net/http"
)

// ID is the unique id type used throughout arla apps
type ID string

// Mutation is an operation submitted to the application (most likely by a user)
// that alters the state of the querystore. Each mutation has a unique ID
// (which is a UUID v1 - therefore also has a rough timestamp encoded). Each
// mutation also contains the ID of the user who submitted the mutation.
type Mutation struct {
	ID     ID
	UserID UserID
	Name   string
	Args   []byte
	Status string
}

// Arg is an argument for a mutation action.
type Arg interface{}

// SafeError is an error that has a way to return a public-facing error message.
// The perpose is to prevent any potentially sensitive infomation from leaking
// out to the public (such as internal details of the system etc).
type SafeError interface {
	SafeError() string
	error
}

// Service is like an http.Handler but with less control over the response.
// It is expected that Serve() always returns JSON data as an io.Reader or an
// SafeError
type Service interface {
	http.Handler
	Serve(r *http.Request) (io.Reader, SafeError)
}
