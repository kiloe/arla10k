package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
)

type TC struct {
	Name    string
	URL     string
	User    *user
	Req     string
	ReqType string
	Res     string
	ResType string
}

type user struct {
	username string
	password string
}

var (
	mutations = `
    {"ID":"", "Name":"exampleOp", "Args":[]}
  `
)

var (
	alice = &user{
		username: "alice",
		password: "%alice123",
	}
	bob = &user{
		username: "bob",
		password: "bobzpasswerd",
	}
	invalid = &user{
		username: "not-a-valid-username",
		password: "not-a-valid-password",
	}
)

var testCases = []*TC{
	&TC{
		Name:    "query members() edge on root",
		URL:     "/query",
		User:    alice,
		ReqType: "application/json",
		Req: `
      members() {
        name
        email
      }
    `,
		ResType: "application/json",
		Res: `{
      "members": [
        {"name":"bob"},
        {"name":"alice"}
      ]
    }`,
	},
}

var server *Server

// compare json bytes a to b
// considered equal if
func cmpjson(a, b []byte) bool {
	return false
}

// try converts a test case into a Request to execute against
// the running server then evaluates if the response if what was
// to be expected
func try(tc *TC) error {
	if tc.ReqType == "" {
		tc.ReqType = "application/json"
	}
	body := strings.NewReader(tc.Req)
	res, err := http.Post("http://localhost"+tc.URL, tc.ReqType, body)
	if err != nil {
		return err
	}
	// check the response type matches
	restype := res.Header.Get("Content-Type")
	if tc.ResType != "" && tc.ResType != restype {
		return fmt.Errorf("expected %s got %s", tc.ResType, restype)
	}
	// read response body
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to parse response body: %s", err)
	}
	if tc.ResType == "application/json" {
		// compare the responses by
		if !cmpjson([]byte(tc.Res), b) {
			return fmt.Errorf("expected response of %s got %s: %s", tc.Res, string(b), err)
		}
	}
	return nil
}

func TestCases(t *testing.T) {
	for _, tc := range testCases {
		if err := try(tc); err != nil {
			t.Fatalf("%s FAILED... %v", tc.Name, err)
		}
	}
}

func TestMain(m *testing.M) {
	server = New(`mysecret`)
	if err := server.Start(); err != nil {
		log.Fatal("failed to start server", err)
	}
	status := m.Run()
	if err := server.Stop(); err != nil {
		log.Fatal("failed to cleanly stop server:", err)
	}
	os.Exit(status)
}
