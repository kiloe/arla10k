package main

import (
	"arla/schema"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

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
	ID       schema.UUID `json:"id"`
	Username string      `json:"username,omitempty"`
	Password string      `json:"password,omitempty"`
	Token    string      `json:"-,omitempty"`
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
		ID:       schema.TimeUUID(),
		Username: "alice",
		Password: "%alice123",
	}
	bob = &user{
		ID:       schema.TimeUUID(),
		Username: "bob",
		Password: "bobzpasswerd",
	}
	kate = &user{
		ID:       schema.TimeUUID(),
		Username: "kate",
		Password: "katington1",
	}
)

func hasKey(k string) func(b []byte) error {
	return func(b []byte) error {
		v := make(map[string]interface{})
		if err := json.Unmarshal(b, &v); err != nil {
			return err
		}
		if t, ok := v[k]; !ok || t == nil {
			return fmt.Errorf("expected response to have key '%s' but got %v", k, string(b))
		}
		return nil
	}
}

var isError = hasKey("error")
var isOK = hasKey("success")

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
		Name:    "register kate and get an access token",
		Method:  POST,
		URL:     "/register",
		Type:    ApplicationJSON,
		Body:    kate.JSON(),
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
		Name:   "dynamically computed property uppername should return BOB",
		Method: POST,
		URL:    "/query",
		User:   bob,
		Type:   TextPlain,
		Body: `
      me(){uppername}
    `,
		ResBody: `{"me":{"uppername":"BOB"}}`,
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
		ResFunc: isOK,
	},

	&TC{
		Name:   "adding same email address for bob again should fail due to unique prop",
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
		ResCode: http.StatusBadRequest,
		ResFunc: isError,
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
		Name:   "update bob's email address",
		Method: POST,
		URL:    "/exec",
		User:   bob,
		Type:   ApplicationJSON,
		Body: `
			{
				"name": "updateEmailAddress",
				"args": ["bob@bob.com", "bob@gmail.com"]
			}
		`,
		ResFunc: isOK,
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
		Name:   "query members().take(1)",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   ApplicationJSON,
		Body: `
      members().take(1) {
        username
      }
    `,
		ResBody: `
			{
				"members": [
					{"username":"alice"}
				]
			}
		`,
	},

	&TC{
		Name:   "query members().slice(0,2)",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   ApplicationJSON,
		Body: `
      members().slice(0,2) {
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

	&TC{
		Name:   "query members().slice(2,1)",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   ApplicationJSON,
		Body: `
      members().slice(2,1) {
        username
      }
    `,
		ResBody: `
			{
				"members": [
					{"username":"kate"}
				]
			}
		`,
	},

	&TC{
		Name:   "query members() on root with email addrs",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   ApplicationJSON,
		Body: `
      members() {
        username
				email_addresses() {
					addr
				}
      }
    `,
		ResBody: `
			{
				"members": [
					{"username":"alice", "email_addresses":[]},
					{"username":"bob", "email_addresses":[{"addr":"bob@gmail.com"}]},
					{"username":"kate","email_addresses":[]}
				]
			}
		`,
	},

	&TC{
		Name:   "give alice a SHOUTY and spacey email",
		Method: POST,
		URL:    "/exec",
		User:   alice,
		Type:   ApplicationJSON,
		Body: `
			{
				"name": "addEmailAddress",
				"args": ["      ALICE@ALICE.com "]
			}
		`,
		ResFunc: isOK,
	},

	&TC{
		Name:   "beforeChange hook should prevent adding an invalid email for alice",
		Method: POST,
		URL:    "/exec",
		User:   alice,
		Type:   ApplicationJSON,
		Body: `
			{
				"name": "addEmailAddress",
				"args": ["not-an-email"]
			}
		`,
		ResType: ApplicationJSON,
		ResCode: http.StatusBadRequest,
		ResFunc: isError,
	},

	&TC{
		Name:   "beforeChange hook should prevent updating to an invalid email for alice",
		Method: POST,
		URL:    "/exec",
		User:   alice,
		Type:   ApplicationJSON,
		Body: `
			{
				"name": "updateEmailAddress",
				"args": ["alice@alice.com", "not-an-email"]
			}
		`,
		ResType: ApplicationJSON,
		ResCode: http.StatusBadRequest,
		ResFunc: isError,
	},

	&TC{
		Name:   "alice should have a single lowercased/trimmed email",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   TextPlain,
		Body: `
      me(){
				email_addresses() {
					addr
				}
			}
    `,
		ResBody: `
			{
				"me":{
					"email_addresses": [{"addr":"alice@alice.com"}]
				}
			}`,
	},

	&TC{
		Name:   "members().pluck(username)",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   ApplicationJSON,
		Body: `
      members().pluck(username)
    `,
		ResBody: `
			{
				"members": ["alice","bob","kate"]
			}
		`,
	},

	&TC{
		Name:   "pluck just the addrs from each member",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   TextPlain,
		Body: `
      members() {
        username
				email_addresses().pluck(addr)
      }
    `,
		ResBody: `
			{
				"members": [
					{"username":"alice", "email_addresses":["alice@alice.com"]},
					{"username":"bob", "email_addresses":["bob@gmail.com"]},
					{"username":"kate","email_addresses":[]}
				]
			}
		`,
	},

	&TC{
		Name:   "members().pluck(email_addresses{addr})",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   ApplicationJSON,
		Body: `
      members().pluck(email_addresses{addr})
    `,
		ResBody: `
			{
				"members": [
					[{"addr":"alice@alice.com"}],
					[{"addr":"bob@gmail.com"}],
					[]
				]
			}
		`,
	},

	&TC{
		Name:   "members().pluck(email_addresses){addr}",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   TextPlain,
		Body: `
      members().pluck(email_addresses){addr}
    `,
		ResBody: `
			{
				"members": [
					[{"addr":"alice@alice.com"}],
					[{"addr":"bob@gmail.com"}],
					[]
				]
			}
		`,
	},

	&TC{
		Name:   "members().pluck(email_addresses.pluck(addr))",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   ApplicationJSON,
		Body: `
      members().pluck(email_addresses.pluck(addr))
    `,
		ResBody: `
			{
				"members": [["alice@alice.com"],["bob@gmail.com"],[]]
			}
		`,
	},

	&TC{
		Name:   "members().pluck(email_addresses).pluck(addr)",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   TextPlain,
		Body: `
      members().pluck(email_addresses).pluck(addr)
    `,
		ResBody: `
			{
				"members": [["alice@alice.com"],["bob@gmail.com"],[]]
			}
		`,
	},

	&TC{
		Name:   "members().pluck(email_addresses).first(){addr}",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   TextPlain,
		Body: `
      members().pluck(email_addresses).first(){addr}
    `,
		ResBody: `
			{
				"members": [{"addr":"alice@alice.com"}]
			}
		`,
	},

	&TC{
		Name:   "members().pluck(email_addresses).pluck(addr).first()",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   TextPlain,
		Body: `
      first_email: members().pluck(email_addresses).pluck(addr).first()
    `,
		ResBody: `
			{
				"first_email": ["alice@alice.com"]
			}
		`,
	},

	&TC{
		Name:   "members().pluck(email_addresses).pluck(addr).count()",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   TextPlain,
		Body: `
      members().pluck(email_addresses).pluck(addr).count()
    `,
		ResBody: `
			{
				"members": 3
			}
		`,
	},

	&TC{
		Name:   "can't pluck from an array of simple types",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   TextPlain,
		Body: `
      members().pluck(email_addresses).pluck(addr).pluck(addr)
    `,
		ResCode: http.StatusBadRequest,
		ResFunc: isError,
	},

	&TC{
		Name:   "simple property types should not accept filters",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   TextPlain,
		Body: `
      members().pluck(email_addresses).pluck(addr.first())
    `,
		ResCode: http.StatusBadRequest,
		ResFunc: isError,
	},

	&TC{
		Name:   "members().pluck(email_addresses.pluck(addr).first())",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   TextPlain,
		Body: `
      members().pluck(email_addresses.pluck(addr).first())
    `,
		ResBody: `
			{
				"members": ["alice@alice.com","bob@gmail.com", null]
			}
		`,
	},

	&TC{
		Name:   "take() from a plucked list",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   TextPlain,
		Body: `
      members().pluck(email_addresses.pluck(addr).first()).take(1)
    `,
		ResBody: `
			{
				"members": ["alice@alice.com"]
			}
		`,
	},

	&TC{
		Name:   "alice should be able to make friends with bob",
		Method: POST,
		URL:    "/exec",
		User:   alice,
		Type:   ApplicationJSON,
		Body: `
			{
				"name": "addFriend",
				"args": ["` + bob.ID.String() + `"]
			}
		`,
		ResFunc: isOK,
	},

	&TC{
		Name:   "bob should not be able to make friends with alice due to unique index",
		Method: POST,
		URL:    "/exec",
		User:   bob,
		Type:   ApplicationJSON,
		Body: `
			{
				"name": "addFriend",
				"args": ["` + alice.ID.String() + `"]
			}
		`,
		ResCode: http.StatusBadRequest,
		ResFunc: isError,
	},

	&TC{
		Name:   "kate should be able to make friends with alice",
		Method: POST,
		URL:    "/exec",
		User:   kate,
		Type:   ApplicationJSON,
		Body: `
			{
				"name": "addFriend",
				"args": ["` + alice.ID.String() + `"]
			}
		`,
		ResFunc: isOK,
	},

	&TC{
		Name:   "alice should see bob and kate as friends",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   TextPlain,
		Body: `
      me(){
				friends() {
					username
				}
			}
    `,
		ResBody: `
			{
				"me":{
					"friends": [{"username":"bob"},{"username":"kate"}]
				}
			}`,
	},

	&TC{
		Name:   "bob should only see alice as a friend",
		Method: POST,
		URL:    "/query",
		User:   bob,
		Type:   TextPlain,
		Body: `
      me(){
				friends() {
					username
				}
			}
    `,
		ResBody: `
			{
				"me":{
					"friends": [{"username":"alice"}]
				}
			}`,
	},

	&TC{
		Name:   "alice wants to be able to fetch just a single friend",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   TextPlain,
		Body: `
      me(){
				friends().first() {
					username
				}
			}
    `,
		ResBody: `
			{
				"me":{
					"friends": {"username":"bob"}
				}
			}`,
	},

	&TC{
		Name:   "alice wants to be able to fetch bob and kate at the same time by aliasing",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   TextPlain,
		Body: `
      me(){
				bob: friends.filter(id="` + bob.ID.String() + `").first() {
					username
				}
				kate: friends.filter(id = '` + kate.ID.String() + `').first() {
					username
				}
			}
    `,
		ResBody: `
			{
				"me":{
					"bob": {"username":"bob"},
					"kate": {"username":"kate"}
				}
			}`,
	},

	&TC{
		Name:   "alice wants to be able to filter her list of friends by id",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   TextPlain,
		Body: `
      me(){
				friends.filter(id = "` + bob.ID.String() + `") {
					id
				}
			}
    `,
		ResBody: `
			{
				"me":{
					"friends": [{"id":"` + bob.ID.String() + `"}]
				}
			}`,
	},

	&TC{
		Name:   "fetch a simple list of numbers",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   TextPlain,
		Body: `
      numbers
    `,
		ResBody: `
			{
				"numbers":[10,5,11]
			}`,
	},

	&TC{
		Name:   "fetch and sort a simple list of numbers",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   TextPlain,
		Body: `
      numbers.sort()
    `,
		ResBody: `
			{
				"numbers":[5,10,11]
			}`,
	},

	&TC{
		Name:   "members().sort(username desc).pluck(username)",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   ApplicationJSON,
		Body: `
      members().sort(username desc).pluck(username)
    `,
		ResBody: `
			{
				"members": ["kate","bob","alice"]
			}
		`,
	},

	&TC{
		Name:   "members().sort(username asc).pluck(username)",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   ApplicationJSON,
		Body: `
      members().sort(username asc).pluck(username)
    `,
		ResBody: `
			{
				"members": ["alice","bob","kate"]
			}
		`,
	},

	&TC{
		Name:   "members().pluck(username).sort(desc)",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   ApplicationJSON,
		Body: `
      members().pluck(username).sort(desc)
    `,
		ResBody: `
			{
				"members": ["kate","bob","alice"]
			}
		`,
	},

	&TC{
		Name:   "alice should see bob as a friend who should see alice as a friend ad infinitum",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   TextPlain,
		Body: `
      me(){
				friends() {
					username
					friends() {
						username
						friends() {
							username
						}
					}
				}
			}
    `,
		ResBody: `{
			"me":{
				"friends":[{
					"username":"bob",
					"friends":[{
						"username":"alice",
						"friends":[{
							"username":"bob"
						},{
							"username":"kate"
						}]
					}]
				},{
					"username":"kate",
					"friends":[{
						"username":"alice",
						"friends":[{
							"username":"bob"
						},{
							"username":"kate"
						}]
					}]
				}]
			}
		}`,
	},

	&TC{
		Name:   "friends can't see friends passwords as it is never in the parent fields",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   TextPlain,
		Body: `
      me(){
				friends() {
					password
				}
			}
    `,
		ResCode: http.StatusBadRequest,
		ResFunc: isError,
	},

	&TC{
		Name:   "beforeDelete hook should prevent alice from getting destroyed",
		Method: POST,
		URL:    "/exec",
		User:   alice,
		Type:   ApplicationJSON,
		Body: `
			{
				"name": "destroyMember",
				"args": []
			}
		`,
		ResType: ApplicationJSON,
		ResCode: http.StatusBadRequest,
		ResFunc: isError,
	},

	&TC{
		Name:   "deleting bob should be fine",
		Method: POST,
		URL:    "/exec",
		User:   bob,
		Type:   ApplicationJSON,
		Body: `
			{
				"name": "destroyMember",
				"args": []
			}
		`,
		ResFunc: isOK,
	},

	&TC{
		Name:   "count() should only show one address remaining due to cascading deletes",
		Method: POST,
		URL:    "/query",
		User:   alice,
		Type:   TextPlain,
		Body: `
			remaining: email_addresses.count()
    `,
		ResBody: `
			{"remaining":1}`,
	},
}

var server *Server

// compare json bytes a to b
func cmpjson(a, b []byte) bool {
	ma := make(map[string]interface{})
	mb := make(map[string]interface{})
	if err := json.Unmarshal(a, &ma); err != nil {
		log.Fatal("FAILED TO UNMARSHAL", a)
		return false
	}
	if err := json.Unmarshal(b, &mb); err != nil {
		log.Fatal("FAILED TO UNMARSHAL", b)
		return false
	}
	return reflect.DeepEqual(ma, mb)
}

// Test converts a test case into a Request to execute against
// the running server then evaluates that the response is what was
// to be expected.
func (tc *TC) Test() (e error) {
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
	defer func() {
		if r := recover(); r != nil {
			e = fail("panic: %v", r)
			panic(r)
		}
	}()
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
		fmt.Fprintln(buf, " ----> Authorization: bearer", tc.User.Token, "("+tc.User.Username+")")
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
			return fail("don't know how to deal with %v", tc.ResType)
		}
	}
	return nil
}

func TestCases(t *testing.T) {
	for _, tc := range testCases {
		fmt.Println("running:", tc.Name, "...")
		if err := tc.Test(); err != nil {
			t.Fatal(err)
		}
		fmt.Println("completed:", tc.Name, ansi.Green, "OK", ansi.Reset)
	}
	// dump successes to screen - makes it less weird since some of the "errors"
	// are actually part of the tests and it can look confusing
	fmt.Print("\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n")
	for _, tc := range testCases {
		fmt.Println(ansi.Green, "PASS", ansi.Reset, tc.Name)
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
	// FIXME: there's a race condition between calling Start() and the HTTP server actually
	// accepting connections ... this only really affects the tests tho.
	time.Sleep(1 * time.Second)
	// run tests
	status := m.Run()
	// shutdown server
	if err := server.Stop(); err != nil {
		log.Fatal("failed to cleanly stop server:", err)
	}
	os.Exit(status)
}
