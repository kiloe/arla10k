package schema

import (
	"io"
	"net/http"
)

// Mutation is an operation submitted to the application (most likely by a user)
// that alters the state of the querystore. Each mutation has a unique ID
// (which is a UUID v1 - therefore also has a rough timestamp encoded). Each
// mutation also contains the ID of the user/account that submitted the mutation.
type Mutation struct {
	ID     UUID          `json:",omitempty"`
	Token  Token         `json:",omitempty"`
	Name   string        `json:",omitempty"`
	Args   []interface{} `json:",omitempty"`
	Status string        `json:",omitempty"`
}

// Arg is an argument for a mutation action.
type Arg interface{}

// Token contains the validated claims that a user/session has.
type Token map[string]string

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
