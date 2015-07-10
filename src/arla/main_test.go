package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/evanphx/json-patch"
	"github.com/mgutz/ansi"
)

const (
	POST = "POST"
	GET  = "GET"
)

const (
	endpoint = "http://localhost"
)

type TC struct {
	// Name is the friendly name of this test case displayed on failure
	Name string
	// Method is the HTTP method (GET/POST)
	Method string
	// URL is the path of the request
	URL string
	// User holds a user to authenticate with
	User *user
	// Body is the request body
	Body string
	// Type is the content-type of request
	Type string
	// ResBody is an optional string representation of the expected response
	ResBody string
	// ResType is the expected response content-type
	ResType string
	// ResCode is the expected response code
	ResCode int
	// ResFunc will be called to verifiy the response if present
	ResFunc func(b []byte) error
}

type user struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Token    string `json:"-,omitempty"`
}

func (u *user) JSON() string {
	b, err := json.Marshal(u)
	if err != nil {
		panic(err)
	}
	return string(b)
}

var (
	mutations = `
    {"ID":"", "Name":"exampleOp", "Args":[]}
  `
)

var (
	alice = &user{
		Username: "alice",
		Password: "%alice123",
	}
	bob = &user{
		Username: "bob",
		Password: "bobzpasswerd",
	}
	invalid = &user{
		Username: "not-a-valid-username",
		Password: "not-a-valid-password",
	}
)

func hasKey(k string) func(b []byte) error {
	return func(b []byte) error {
		v := make(map[string]interface{})
		if err := json.Unmarshal(b, &v); err != nil {
			return err
		}
		if t, ok := v[k]; !ok || t == nil {
			return fmt.Errorf("expected response to have key '%s' access_token got %v", k, string(b))
		}
		return nil
	}
}

var testCases = []*TC{

	&TC{
		Name:    "register alice and get an access token",
		Method:  POST,
		URL:     "/register",
		Type:    ApplicationJSON,
		Body:    alice.JSON(),
		ResFunc: hasKey("access_token"),
	},

	&TC{
		Name:    "register bob and get an access token",
		Method:  POST,
		URL:     "/register",
		Type:    ApplicationJSON,
		Body:    bob.JSON(),
		ResFunc: hasKey("access_token"),
	},

	&TC{
		Name:   "query me() on root for alice",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   TextPlain,
		Body: `
      me() {
        username
      }
    `,
		ResBody: `{"me":{"username":"alice"}}`,
	},

	&TC{
		Name:   "query me() on root for bob",
		Method: POST,
		URL:    "/query",
		User:   bob,
		Type:   TextPlain,
		Body: `
      me(){username}
    `,
		ResBody: `{"me":{"username":"bob"}}`,
	},

	&TC{
		Name:   "add an email address for bob",
		Method: POST,
		URL:    "/exec",
		User:   bob,
		Type:   ApplicationJSON,
		Body: `
			{
				"name": "addEmailAddress",
				"args": ["bob@bob.com"]
			}
		`,
		ResFunc: hasKey("success"),
	},

	&TC{
		Name:   "bob should have an email address",
		Method: POST,
		URL:    "/query",
		User:   bob,
		Type:   TextPlain,
		Body: `
      me(){
				username
				email_addresses() {
					addr
				}
			}
    `,
		ResBody: `
			{
				"me":{
					"username":"bob",
					"email_addresses": [
						{"addr": "bob@bob.com"}
					]
				}
			}`,
	},

	&TC{
		Name:   "alice should NOT have any email addresses",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   TextPlain,
		Body: `
      me(){
				username
				email_addresses() {
					addr
				}
			}
    `,
		ResBody: `
			{
				"me":{
					"username":"alice",
					"email_addresses": []
				}
			}`,
	},

	&TC{
		Name:   "query members() on root",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   ApplicationJSON,
		Body: `
      members() {
        username
      }
    `,
		ResBody: `
			{
				"members": [
					{"username":"alice"},
					{"username":"bob"}
				]
			}
		`,
	},
}

var server *Server

// compare json bytes a to b
// considered equal if
func cmpjson(a, b []byte) bool {
	return jsonpatch.Equal(a, b)
}

// Test converts a test case into a Request to execute against
// the running server then evaluates that the response is what was
// to be expected.
func (tc *TC) Test() error {
	buf := &bytes.Buffer{}
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "+------------------------------------------")
	fmt.Fprintln(buf, "| ", tc.Name)
	fmt.Fprintln(buf, "+------------------------------------------")
	fmt.Fprintln(buf)
	fail := func(msg string, args ...interface{}) error {
		fmt.Fprint(buf, "\n", ansi.Red)
		fmt.Fprintf(buf, msg, args...)
		fmt.Fprint(buf, "\n", ansi.Reset)
		return errors.New(buf.String())
	}
	if tc.Type == "" {
		tc.Type = ApplicationJSON
	}
	// authenticate
	if tc.User != nil {
		if tc.User.Token == "" {
			authb, err := json.Marshal(tc.User)
			if err != nil {
				return fail("failed to marshal user during authentication: %v", err)
			}
			fmt.Fprintln(buf, " ----> ", "POST /authenticate")
			fmt.Fprintln(buf, " ----> ", string(authb))
			auth := bytes.NewReader(authb)
			res, err := http.Post(endpoint+"/authenticate", ApplicationJSON, auth)
			if err != nil {
				return fail("failed to authenticate: %v", err)
			}
			b, err := ioutil.ReadAll(res.Body)
			if err != nil {
				return fail("failed to read authentication body: %v", err)
			}
			t := make(map[string]string)
			json.Unmarshal(b, &t)
			var ok bool
			tc.User.Token, ok = t["access_token"]
			if !ok {
				return fail("expected to get an access token but got: %v", string(b))
			}
			fmt.Fprintln(buf, " <---- ", string(b))
			fmt.Fprintln(buf)
		}
	}
	// build request
	body := strings.NewReader(tc.Body)
	req, err := http.NewRequest(tc.Method, endpoint+tc.URL, body)
	if err != nil {
		return fail("failed to build request: %v", err)
	}
	fmt.Fprintln(buf, " ---->", tc.Method, tc.URL)
	fmt.Fprintln(buf, " ----> Content-Type:", tc.Type)
	if tc.User != nil && tc.User.Token != "" {
		fmt.Fprintln(buf, " ----> Authorization: bearer", tc.User.Token)
		req.Header.Add("Authorization", "bearer "+tc.User.Token)
	}
	fmt.Fprintln(buf, " ----> ", tc.Body)
	var c http.Client
	res, err := c.Do(req)
	if err != nil {
		return fail("failed to make HTTP POST: %v", err)
	}
	// read response body
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fail("failed to parse response body: %v", err)
	}
	if b == nil {
		return fail("failed to parse response body: b was nil?")
	}
	resBody := string(b)
	resType := res.Header.Get("Content-Type")
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, " <---- Status:", res.StatusCode)
	fmt.Fprintln(buf, " <---- Content-Type:", resType)
	fmt.Fprintln(buf, " <----", resBody)
	// check response code
	if tc.ResCode == 0 {
		tc.ResCode = http.StatusOK
	}
	if tc.ResCode != res.StatusCode {
		return fail("expected status code to be %v", tc.ResCode)
	}
	// check the response type matches
	if resType == "" {
		return fail("expected response to have a Content-Type")
	}
	if tc.ResType == "" {
		tc.ResType = ApplicationJSON
	}
	if tc.ResType != resType {
		return fail("response Content-Type to be %v", tc.ResType)
	}
	if tc.ResFunc != nil {
		if err := tc.ResFunc(b); err != nil {
			return fail("failure during ResFunc: %v", err)
		}
	} else {
		switch tc.ResType {
		case ApplicationJSON:
			if !cmpjson([]byte(tc.ResBody), b) {
				return fail("expected json response to be:\n%v", tc.ResBody)
			}
		case TextPlain:
			if tc.ResBody != string(b) {
				return fail("expected text response to be:\n%v", tc.ResBody)
			}
		default:
			panic("don't know how to deal with " + tc.ResType + " responses")
		}
	}
	return nil
}

func TestCases(t *testing.T) {
	for _, tc := range testCases {
		if err := tc.Test(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMain(m *testing.M) {
	// start server
	server = New(Config{
		ConfigPath: "/app/test-app/index.js",
		DataDir:    "/tmp/",
		Secret:     "mysecret",
		Debug:      true,
	})
	if err := server.Start(); err != nil {
		log.Fatal("failed to start server", err)
	}
	// run tests
	status := m.Run()
	// shutdown server
	if err := server.Stop(); err != nil {
		log.Fatal("failed to cleanly stop server:", err)
	}
	os.Exit(status)
}
