package main

import (
	"arla/schema"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
)

var tests = make([]*TestCase, 0)

func req(url string, data interface{}, token string) *http.Response {
	body, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	fmt.Println(" ---->", "POST", url)
	r := bytes.NewReader(body)
	req, err := http.NewRequest("POST", "http://localhost"+url, r)
	if err != nil {
		panic("failed to build request")
	}
	fmt.Println(" ----> Content-Type:", ApplicationJSON)
	req.Header.Set("Content-Type", ApplicationJSON)
	if token != "" {
		fmt.Println(" ----> Authorization: bearer", token)
		req.Header.Add("Authorization", "bearer "+token)
	}
	fmt.Println(" ----> ", string(body))
	var c http.Client
	res, err := c.Do(req)
	if err != nil {
		panic("failed to make HTTP POST")
	}
	return res
}

type TestCase struct {
	// API url
	URL string
	// User holds a user to authenticate with
	User *User
	// Query the query to send
	Data interface{}
	// Check will be called to verifiy the response if present
	Checks []func() error
	// response fields
	res        *http.Response
	resMap     map[string]interface{}
	resString  string
	shouldFail bool
}

func (tc *TestCase) Test() (err error) {
	tc.res = req(tc.URL, tc.Data, tc.User.Token)
	resBody, err := ioutil.ReadAll(tc.res.Body)
	if err != nil {
		return err
	}
	tc.resString = string(resBody)
	tc.resMap = make(map[string]interface{})
	if err := json.Unmarshal(resBody, &tc.resMap); err != nil {
		return err
	}
	if len(tc.Checks) == 0 {
		return errors.New("TestCase has no checks set")
	}
	if tc.shouldFail {
		if tc.res.StatusCode == http.StatusOK {
			return fmt.Errorf("expected request to fail but got 200 OK and response: %v", tc.resString)
		}
	} else {
		if tc.res.StatusCode != http.StatusOK {
			return fmt.Errorf("expected 200 OK response but got %d: %v", tc.res.StatusCode, tc.resString)
		}
	}
	tc.ShouldBeJSON()
	for _, check := range tc.Checks {
		if err := check(); err != nil {
			return err
		}
	}
	return nil
}

func (tc *TestCase) ShouldReturn(jsonResponse string) *TestCase {
	tc.Checks = append(tc.Checks, func() error {
		expectedMap := make(map[string]interface{})
		if err := json.Unmarshal([]byte(jsonResponse), &expectedMap); err != nil {
			return fmt.Errorf("invalid json in test case- ie. the test itself is broken")
		}
		if !reflect.DeepEqual(tc.resMap, expectedMap) {
			return fmt.Errorf("expected json response to be: %s\nbut got: %s\n", jsonResponse, tc.resString)
		}
		return nil
	})
	return tc
}

func (tc *TestCase) ShouldBeJSON() *TestCase {
	tc.Checks = append(tc.Checks, func() error {
		ct := tc.res.Header.Get("Content-Type")
		if ct != ApplicationJSON {
			return fmt.Errorf("expected Content-Type to be %s got: %s", ApplicationJSON, ct)
		}
		return nil
	})
	return tc
}

// ShouldFail changes the testcase to expect a non 200 response
// and looks for an "error" key in response
func (tc *TestCase) ShouldFail() *TestCase {
	tc.shouldFail = true
	tc.Checks = append(tc.Checks, func() error {
		k := "error"
		if t, ok := tc.resMap[k]; !ok || t == nil {
			return fmt.Errorf("expected response to have key '%s' but got %v", k, tc.resString)
		}
		return nil
	})
	return tc
}

// Should succeed looks for a "success" key in the response
func (tc *TestCase) ShouldSucceed() *TestCase {
	tc.shouldFail = false
	tc.Checks = append(tc.Checks, func() error {
		k := "success"
		if t, ok := tc.resMap[k]; !ok || t == nil {
			return fmt.Errorf("expected response to have key '%s' but got %v", k, tc.resString)
		}
		return nil
	})
	return tc
}

// ShouldBeAuthenticated checks the response has an access_token
// It also stores the access_token on the user - so this function must be
// called in order to make future requests ... yeah yeah it's weird :P
func (tc *TestCase) ShouldBeAuthenticated() *TestCase {
	tc.Checks = append(tc.Checks, func() error {
		k := "access_token"
		t, ok := tc.resMap[k]
		if !ok || t == nil {
			return fmt.Errorf("expected response to have key '%s' but got %v", k, tc.resString)
		}
		tc.User.Token, ok = t.(string)
		if !ok {
			return fmt.Errorf("expected access_token to be a string got %v", tc.resString)
		}
		return nil
	})
	return tc
}

type User struct {
	ID       schema.UUID `json:"id"`
	Name     string      `json:"name,omitempty"`
	Username string      `json:"username,omitempty"`
	Password string      `json:"password,omitempty"`
	Token    string      `json:"-,omitempty"`
}

// Query starts a /query request
func (u *User) Query(q string, args ...interface{}) *TestCase {
	tc := &TestCase{
		URL:  "/query",
		User: u,
		Data: &schema.Query{
			Query: q,
			Args:  args,
		},
	}
	tests = append(tests, tc)
	return tc
}

// Exec starts a /exec request
func (u *User) Exec(name string, args ...interface{}) *TestCase {
	tc := &TestCase{
		URL:  "/exec",
		User: u,
		Data: &schema.Mutation{
			Name: name,
			Args: args,
		},
	}
	tests = append(tests, tc)
	return tc
}

// Register attempts to sign up a user
func (u *User) Register() *TestCase {
	tc := &TestCase{
		URL:  "/register",
		User: u,
		Data: u,
	}
	tests = append(tests, tc)
	return tc
}

// Authenticate requsts an access token
func (u *User) Authenticate() *TestCase {
	tc := &TestCase{
		URL:  "/authenticate",
		User: u,
		Data: u,
	}
	tests = append(tests, tc)
	return tc
}

// JSON stringifies the user
func (u *User) JSON() string {
	b, err := json.Marshal(u)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func NewUser(name, pw string) *User {
	return &User{
		ID:       schema.TimeUUID(),
		Username: name,
		Name:     name,
		Password: pw,
	}
}
