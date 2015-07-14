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
	"regexp"
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
	// Query is shorthand for setting a Query test case
	Query     string
	QueryArgs []interface{}
	// Check will be called to verifiy the response if present
	Check Check
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

type Check func(b []byte, res *http.Response) error

func hasKey(k string, b []byte) error {
	v := make(map[string]interface{})
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	if t, ok := v[k]; !ok || t == nil {
		return fmt.Errorf("expected response to have key '%s' but got %v", k, string(b))
	}
	return nil
}

func isError() Check {
	return func(b []byte, res *http.Response) error {
		if res.StatusCode != http.StatusBadRequest {
			return fmt.Errorf("expected status code %d", http.StatusBadRequest)
		}
		return hasKey("error", b)
	}
}

func isOK() Check {
	return func(b []byte, res *http.Response) error {
		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("expected status code %d", http.StatusOK)
		}
		return hasKey("success", b)
	}
}

func isAuthenticated() Check {
	return func(b []byte, res *http.Response) error {
		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("expected status code %d", http.StatusOK)
		}
		return hasKey("access_token", b)
	}
}

func isJSON(a string) Check {
	return func(b []byte, res *http.Response) error {
		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("expected status code %d", http.StatusOK)
		}
		ma := make(map[string]interface{})
		mb := make(map[string]interface{})
		if err := json.Unmarshal([]byte(a), &ma); err != nil {
			return err
		}
		if err := json.Unmarshal(b, &mb); err != nil {
			return err
		}
		if !reflect.DeepEqual(ma, mb) {
			return fmt.Errorf("expected json response to be:\n%s", a)
		}
		return nil

	}
}

var testCases = []*TC{

	&TC{
		Name:  "register alice and get an access token",
		URL:   "/register",
		Body:  alice.JSON(),
		Check: isAuthenticated(),
	},

	&TC{
		Name:  "register bob and get an access token",
		URL:   "/register",
		Body:  bob.JSON(),
		Check: isAuthenticated(),
	},

	&TC{
		Name:  "register kate and get an access token",
		URL:   "/register",
		Body:  kate.JSON(),
		Check: isAuthenticated(),
	},

	&TC{
		User: alice,
		Query: `
      me() {
        username
      }
    `,
		Check: isJSON(`{"me":{"username":"alice"}}`),
	},

	&TC{
		Query: `me(){username}`,
		User:  bob,
		Check: isJSON(`{"me":{"username":"bob"}}`),
	},

	&TC{
		Query: `me(){uppername}`,
		User:  bob,
		Check: isJSON(`{"me":{"uppername":"BOB"}}`),
	},

	&TC{
		Name: "add an email address for bob",
		URL:  "/exec",
		User: bob,
		Body: `
			{
				"name": "addEmailAddress",
				"args": ["bob@bob.com"]
			}
		`,
		Check: isOK(),
	},

	&TC{
		Name: "adding same email address for bob again should fail due to unique prop",
		URL:  "/exec",
		User: bob,
		Body: `
			{
				"name": "addEmailAddress",
				"args": ["bob@bob.com"]
			}
		`,
		Check: isError(),
	},

	&TC{
		Query: `
      me(){
				username
				email_addresses() {
					addr
				}
			}
    `,
		User: bob,
		Check: isJSON(`{
			"me":{
				"username":"bob",
				"email_addresses": [
					{"addr": "bob@bob.com"}
				]
			}
		}`),
	},

	&TC{
		Name: "update bob's email address",
		URL:  "/exec",
		User: bob,
		Body: `
			{
				"name": "updateEmailAddress",
				"args": ["bob@bob.com", "bob@gmail.com"]
			}
		`,
		Check: isOK(),
	},

	&TC{
		User: alice,
		Query: `
      me(){
				username
				email_addresses() {
					addr
				}
			}
    `,
		Check: isJSON(`
			{
				"me":{
					"username":"alice",
					"email_addresses": []
				}
			}`),
	},

	&TC{
		User: alice,
		Query: `
      members().take(1) {
        username
      }
    `,
		Check: isJSON(`
			{
				"members": [
					{"username":"alice"}
				]
			}
		`),
	},

	&TC{
		User: alice,
		Query: `
      members().slice(0,2) {
        username
      }
    `,
		Check: isJSON(`
			{
				"members": [
					{"username":"alice"},
					{"username":"bob"}
				]
			}
		`),
	},

	&TC{
		User: alice,
		Query: `
      members().slice(2,1) {
        username
      }
    `,
		Check: isJSON(`
			{
				"members": [
					{"username":"kate"}
				]
			}
		`),
	},

	&TC{
		User: alice,
		Query: `
      members() {
        username
				email_addresses() {
					addr
				}
      }
    `,
		Check: isJSON(`
			{
				"members": [
					{"username":"alice", "email_addresses":[]},
					{"username":"bob", "email_addresses":[{"addr":"bob@gmail.com"}]},
					{"username":"kate","email_addresses":[]}
				]
			}
		`),
	},

	&TC{
		Name: "give alice a SHOUTY and spacey email",
		URL:  "/exec",
		User: alice,
		Body: `
			{
				"name": "addEmailAddress",
				"args": ["      ALICE@ALICE.com "]
			}
		`,
		Check: isOK(),
	},

	&TC{
		Name: "beforeChange hook should prevent adding an invalid email for alice",
		URL:  "/exec",
		User: alice,
		Body: `
			{
				"name": "addEmailAddress",
				"args": ["not-an-email"]
			}
		`,
		Check: isError(),
	},

	&TC{
		Name: "beforeChange hook should prevent updating to an invalid email for alice",
		URL:  "/exec",
		User: alice,
		Body: `
			{
				"name": "updateEmailAddress",
				"args": ["alice@alice.com", "not-an-email"]
			}
		`,
		Check: isError(),
	},

	&TC{
		User: alice,
		Query: `
      me(){
				email_addresses() {
					addr
				}
			}
    `,
		Check: isJSON(`
			{
				"me":{
					"email_addresses": [{"addr":"alice@alice.com"}]
				}
			}`),
	},

	&TC{
		User: alice,
		Query: `
      members().pluck(username)
    `,
		Check: isJSON(`
			{
				"members": ["alice","bob","kate"]
			}
		`),
	},

	&TC{
		User: alice,
		Query: `
      members() {
        username
				email_addresses().pluck(addr)
      }
    `,
		Check: isJSON(`
			{
				"members": [
					{"username":"alice", "email_addresses":["alice@alice.com"]},
					{"username":"bob", "email_addresses":["bob@gmail.com"]},
					{"username":"kate","email_addresses":[]}
				]
			}
		`),
	},

	&TC{
		User: alice,
		Query: `
      members().pluck(email_addresses{addr})
    `,
		Check: isJSON(`
			{
				"members": [
					[{"addr":"alice@alice.com"}],
					[{"addr":"bob@gmail.com"}],
					[]
				]
			}
		`),
	},

	&TC{
		User: alice,
		Query: `
      members().pluck(email_addresses){addr}
    `,
		Check: isJSON(`
			{
				"members": [
					[{"addr":"alice@alice.com"}],
					[{"addr":"bob@gmail.com"}],
					[]
				]
			}
		`),
	},

	&TC{
		User: alice,
		Query: `
      members().pluck(email_addresses.pluck(addr))
    `,
		Check: isJSON(`
			{
				"members": [["alice@alice.com"],["bob@gmail.com"],[]]
			}
		`),
	},

	&TC{
		User: alice,
		Query: `
      members().pluck(email_addresses).pluck(addr)
    `,
		Check: isJSON(`
			{
				"members": [["alice@alice.com"],["bob@gmail.com"],[]]
			}
		`),
	},

	&TC{
		User: alice,
		Query: `
      members().pluck(email_addresses).first(){addr}
    `,
		Check: isJSON(`
			{
				"members": [{"addr":"alice@alice.com"}]
			}
		`),
	},

	&TC{
		User: alice,
		Query: `
      first_email: members().pluck(email_addresses).pluck(addr).first()
    `,
		Check: isJSON(`
			{
				"first_email": ["alice@alice.com"]
			}
		`),
	},

	&TC{
		User: alice,
		Query: `
      members().pluck(email_addresses).pluck(addr).count()
    `,
		Check: isJSON(`
			{
				"members": 3
			}
		`),
	},

	&TC{
		User: alice,
		Query: `
      members().pluck(email_addresses).pluck(addr).pluck(addr)
    `,
		Check: isError(),
	},

	&TC{
		User: alice,
		Query: `
      members().pluck(email_addresses).pluck(addr.first())
    `,
		Check: isError(),
	},

	&TC{
		User: alice,
		Query: `
      members().pluck(email_addresses.pluck(addr).first())
    `,
		Check: isJSON(`
			{
				"members": ["alice@alice.com","bob@gmail.com", null]
			}
		`),
	},

	&TC{
		User: alice,
		Query: `
      members().pluck(email_addresses.pluck(addr).first()).take(1)
    `,
		Check: isJSON(`
			{
				"members": ["alice@alice.com"]
			}
		`),
	},

	&TC{
		Name: "add bob as alice's friend",
		URL:  "/exec",
		User: alice,
		Body: `
			{
				"name": "addFriend",
				"args": ["` + bob.ID.String() + `"]
			}
		`,
		Check: isOK(),
	},

	&TC{
		Name: "bob should not be able to make friends with alice due to unique index",
		URL:  "/exec",
		User: bob,
		Body: `
			{
				"name": "addFriend",
				"args": ["` + alice.ID.String() + `"]
			}
		`,
		Check: isError(),
	},

	&TC{
		Name: "kate should be able to make friends with alice",
		URL:  "/exec",
		User: kate,
		Body: `
			{
				"name": "addFriend",
				"args": ["` + alice.ID.String() + `"]
			}
		`,
		Check: isOK(),
	},

	&TC{
		User: alice,
		Query: `
      me(){
				friends() {
					username
				}
			}
    `,
		Check: isJSON(`
			{
				"me":{
					"friends": [{"username":"bob"},{"username":"kate"}]
				}
			}`),
	},

	&TC{
		User: bob,
		Query: `
      me(){
				friends() {
					username
				}
			}
    `,
		Check: isJSON(`
			{
				"me":{
					"friends": [{"username":"alice"}]
				}
			}`),
	},

	&TC{
		User: alice,
		Query: `
      me(){
				friends().first() {
					username
				}
			}
    `,
		Check: isJSON(`
			{
				"me":{
					"friends": {"username":"bob"}
				}
			}`),
	},

	&TC{
		User: alice,
		Query: `
      me(){
				bob: friends.filter(id="` + bob.ID.String() + `").first() {
					username
				}
				kate: friends.filter(id = '` + kate.ID.String() + `').first() {
					username
				}
			}
    `,
		Check: isJSON(`
			{
				"me":{
					"bob": {"username":"bob"},
					"kate": {"username":"kate"}
				}
			}`),
	},

	&TC{
		User: alice,
		Query: `
      me(){
				friends.filter(id = "` + bob.ID.String() + `") {
					id
				}
			}
    `,
		Check: isJSON(`
			{
				"me":{
					"friends": [{"id":"` + bob.ID.String() + `"}]
				}
			}`),
	},

	&TC{
		User: alice,
		Query: `
      numbers
    `,
		Check: isJSON(`
			{
				"numbers":[10,5,11]
			}`),
	},

	&TC{
		User: alice,
		Query: `
      numbers.sort()
    `,
		Check: isJSON(`
			{
				"numbers":[5,10,11]
			}`),
	},

	&TC{
		User: alice,
		Query: `
      members().sort(username desc).pluck(username)
    `,
		Check: isJSON(`
			{
				"members": ["kate","bob","alice"]
			}
		`),
	},

	&TC{
		User: alice,
		Query: `
      members().sort(username asc).pluck(username)
    `,
		Check: isJSON(`
			{
				"members": ["alice","bob","kate"]
			}
		`),
	},

	&TC{
		User: alice,
		Query: `
      members().pluck(username).sort(desc)
    `,
		Check: isJSON(`
			{
				"members": ["kate","bob","alice"]
			}
		`),
	},

	&TC{
		User: alice,
		Query: `
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
		Check: isJSON(`{
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
		}`),
	},

	&TC{
		User: alice,
		Query: `
      me(){
				friends() {
					password
				}
			}
    `,
		Check: isError(),
	},

	&TC{
		Name:   "beforeDelete hook should prevent alice from getting destroyed",
		Method: POST,
		URL:    "/exec",
		User:   alice,
		Body: `
			{
				"name": "destroyMember",
				"args": []
			}
		`,
		Check: isError(),
	},

	&TC{
		Name:   "deleting bob should be fine",
		Method: POST,
		URL:    "/exec",
		User:   bob,
		Body: `
			{
				"name": "destroyMember",
				"args": []
			}
		`,
		Check: isOK(),
	},

	&TC{
		User: alice,
		Query: `
			remaining: email_addresses.count()
    `,
		Check: isJSON(`{"remaining":1}`),
	},
}

var server *Server

// Test converts a test case into a Request to execute against
// the running server then evaluates that the response is what was
// to be expected.
func (tc *TC) Test() (e error) {
	// query shorthand
	if tc.Query != "" {
		tc.URL = "/query"
		re := regexp.MustCompile(`[\s\n]+`)
		tc.Name = strings.TrimSpace(re.ReplaceAllString(tc.Query, " "))
		q := &struct {
			Query string `json:"query"`
			Args  []interface{}
		}{
			Query: tc.Query,
			Args:  tc.QueryArgs,
		}
		b, err := json.Marshal(q)
		if err != nil {
			panic("failed to marshal tc.Query")
		}
		tc.Body = string(b)
	}
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
	req.Header.Set("Content-Type", ApplicationJSON)
	fmt.Fprintln(buf, " ---->", tc.Method, tc.URL)
	fmt.Fprintln(buf, " ----> Content-Type:", ApplicationJSON)
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
	// check the response type matches
	if resType == "" {
		return fail("expected response to have a Content-Type")
	}
	if resType != ApplicationJSON {
		return fail("response Content-Type to be %v", ApplicationJSON)
	}
	if tc.Check == nil {
		return fail("no Check for test")
	}
	if err := tc.Check(b, res); err != nil {
		return fail("%v", err)
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
