package main

import (
	"arla/schema"
	"encoding/json"
	"net/http"
	"strings"
	"errors"
)

import "github.com/jackc/pgx"

// Error type used my HandleFunc
type Error struct {
	err     error
	code    int
	Message string `json:"error"`
	// QueryError fields
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	Offset   int    `json:"offset,omitempty"`
	Context  string `json:"context,omitempty"`
	Property string `json:"property,omitempty"`
	Type     string `json:"type,omitempty"`
	Kind     string `json:"kind,omitempty"`
	// MutationError fields
	Mutation *schema.Mutation `json:"mutation,omitempty"`
}

func (e *Error) Error() string {
	return e.err.Error()
}

// userError wraps an error with a 400 status and filters which messages
// get seen my the user
func userError(err error) *Error {
	e := &Error{
		err:     err,
		code:    http.StatusBadRequest,
		Message: "there was a problem processing your request",
	}
	if pgerr, ok := err.(pgx.PgError); ok {
		// strip supurfluous Error that gets added via plv8 somewhere
		if strings.HasPrefix(pgerr.Message, "Error: ") {
			pgerr.Message = strings.Replace(pgerr.Message, "Error: ", "", 1)
		}
		// Detect errors with extended info
		if strings.HasPrefix(pgerr.Message, "UserError:") {
			e.Message = strings.Replace(pgerr.Message, "UserError: ", "", 1)
		} else if strings.HasPrefix(pgerr.Message, "QueryError:") {
			b := []byte(strings.Replace(pgerr.Message, "QueryError: ", "", 1))
			json.Unmarshal(b, &e)
		} else if strings.HasPrefix(pgerr.Message, "MutationError:") {
			b := []byte(strings.Replace(pgerr.Message, "MutationError: ", "", 1))
			json.Unmarshal(b, &e)
		}
	}
	return e
}

// internalError wraps an error with a 500 status and masks the
// actual error message from the end-user completely
func internalError(err error) *Error {
	return &Error{
		err:     err,
		code:    http.StatusInternalServerError,
		Message: "a server error prevented your request from being processed correctly",
	}
}

// authError wraps an error with a 401 error and masks the
// actual error message from the end-user completely
func authError(err error) *Error {
	return &Error{
		err:     err,
		code:    http.StatusUnauthorized,
		Message: "you are not authorized to perform this request",
	}
}

// If a request is in the middle of being processed when server is
// shutdown or when qs or ms fails then return a "come back later" error
func tempError() *Error {
	return &Error{
		err: errors.New("system temporarily offline"),
		code: http.StatusServiceUnavailable,
		Message: "Service is temporarily unavailable",
	}
}
