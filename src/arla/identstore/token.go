package ident

import (
	"crypto/rand"
	"encoding/base64"
)

const (
	tokenSize = 48
)

type Token string

func NewToken() (tok Token, err error) {
	b := make([]byte, tokenSize)
	if _, err = rand.Read(b); err != nil {
		return
	}
	tok = Token(base64.URLEncoding.EncodeToString(b))
	return
}
