package main

import (
	"encoding/json"
	"net/http"
	"strings"
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
		if strings.HasPrefix(pgerr.Message, "UserError:") {
			e.Message = strings.Replace(pgerr.Message, "UserError: ", "", 1)
		}
		if strings.HasPrefix(pgerr.Message, "QueryError:") {
			b := []byte(strings.Replace(pgerr.Message, "QueryError: ", "", 1))
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
