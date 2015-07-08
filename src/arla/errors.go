package main

import (
	"fmt"
	"net/http"
)

// Error type used my HandleFunc
type Error struct {
	err     error
	code    int
	Message string
}

// Error implements the error interface
func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s [%v]", e.Message, e.err.Error(), e.code)
}

// userError wraps an error with a 400 status and filters which messages
// get seen my the user
func userError(err error) *Error {
	return &Error{
		err:     err,
		code:    http.StatusBadRequest,
		Message: "there was a problem processing your request",
	}
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
