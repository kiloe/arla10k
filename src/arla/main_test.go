package main

import (
	"arla/schema"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mgutz/ansi"
)

// create our test users
var (
	alice = NewUser("alice", "%alice123")
	bob   = NewUser("bob", "bobzpasswerd")
	kate  = NewUser("kate", "katington1")
)

func TestAPI(t *testing.T) {

	// alice should already exist (created by loading the mutation log)
	alice.Authenticate().ShouldBeAuthenticated()

	// register bob
	bob.Register().ShouldBeAuthenticated()

	// kate shouldn't be able to login as she doesn't exist yet
	kate.Authenticate().ShouldFail()

	// register kate and login
	kate.Register().ShouldBeAuthenticated()
	kate.Authenticate().ShouldBeAuthenticated()

	// fetch our own username
	alice.Query(`me(){username}`).ShouldReturn(`
		{"me":{"username":"alice"}}
	`)

	// check weird "name_" prop workaround converts to "name"
	bob.Query(`me(){name}`).ShouldReturn(`
		{"me":{"name":"bob"}}
	`)

	// fetch name via dynamic property
	bob.Query(`me(){uppername}`).ShouldReturn(`
		{"me":{"uppername":"BOB"}}
	`)

	// execute mutation to add an email_address record
	bob.Exec("addEmailAddress", "83d6e614-b12c-4a24-b1a3-efbd35c34ad8", "bob@bob.com").ShouldSucceed()
	bob.Exec("addEmailAddress", "55e33341-7c2d-4e8e-a01b-f45abd9ab57c", "bob@bob.com").ShouldFail() // already exists

	// fetch email_address record via property
	bob.Query(`
		me(){
			username
			email_addresses() {
				addr
			}
		}
	`).ShouldReturn(`
		{
			"me":{
				"username":"bob",
				"email_addresses": [
					{"addr": "bob@bob.com"}
				]
			}
		}
	`)

	// fetch email_address_plus record via property
	bob.Query(`
		me(){
			username
			email_addresses_plus() {
				fake
				addr
			}
		}
	`).ShouldReturn(`
		{
			"me":{
				"username":"bob",
				"email_addresses_plus": [
					{"fake":true, "addr": "bob@bob.com"}
				]
			}
		}
	`)

	// execute mutation to change email address
	bob.Exec("updateEmailAddress", "bob@bob.com", "bob@gmail.com").ShouldSucceed()

	// ...check email addresses are as expected
	alice.Query(`
		members() {
			username
			email_addresses() {
				addr
			}
		}
	`).ShouldReturn(`
		{
			"members": [
				{"username":"alice", "email_addresses":[]},
				{"username":"bob", "email_addresses":[{"addr":"bob@gmail.com"}]},
				{"username":"kate","email_addresses":[]}
			]
		}
	`)

	// ensure alice doesn't have any email addresses yet
	alice.Query(`
		me(){
			username
			email_addresses() {
				addr
			}
		}
	`).ShouldReturn(`
		{
			"me":{
				"username":"alice",
				"email_addresses": []
			}
		}
	`)

	// beforeChange hook should cause email addr to get lowercased
	alice.Exec("addEmailAddress", "da4b8dcc-b46d-422c-8c30-b8f9ff375c43", "      ALICE@ALICE.com ").ShouldSucceed()

	// beforeChange hooks should prevent invalid emails
	alice.Exec("addEmailAddress", "f7f44141-6e23-417b-abdd-5d1f840f97db", "not-an-email").ShouldFail()
	alice.Exec("updateEmailAddress", "alice@alice.com", "not-an-email").ShouldFail()

	// alice should just have a single (lowercase) email
	alice.Query(`
		me(){
			email_addresses() {
				addr
			}
		}
	`).ShouldReturn(`
		{
			"me":{
				"email_addresses": [{"addr":"alice@alice.com"}]
			}
		}
	`)

	// addrs should have been precomputed via the afterChange trigger on email
	alice.Query(`
		me(){addrs}
	`).ShouldReturn(`
		{
			"me":{
				"addrs": ["alice@alice.com"]
			}
		}
	`)

	// ----------------------------------------

	// pluck should allow pulling a single property from members
	alice.Query(`members().pluck(username)`).ShouldReturn(`
		{
			"members": ["alice","bob","kate"]
		}
	`)

	alice.Query(`
		members() {
			username
			email_addresses().pluck(addr)
		}
	`).ShouldReturn(`
		{
			"members": [
				{"username":"alice", "email_addresses":["alice@alice.com"]},
				{"username":"bob", "email_addresses":["bob@gmail.com"]},
				{"username":"kate","email_addresses":[]}
			]
		}
	`)

	// pluck should work on properties that return arrays of objects
	alice.Query(`members().pluck(email_addresses{addr})`).ShouldReturn(`
		{
			"members": [
				[{"addr":"alice@alice.com"}],
				[{"addr":"bob@gmail.com"}],
				[]
			]
		}
	`)

	// placing the property list after the entire propety should be equivilent
	alice.Query(`members().pluck(email_addresses){addr}`).ShouldReturn(`
		{
			"members": [
				[{"addr":"alice@alice.com"}],
				[{"addr":"bob@gmail.com"}],
				[]
			]
		}
	`)

	// should be possible to nest pluck statements
	alice.Query(`members().pluck(email_addresses.pluck(addr))`).ShouldReturn(`
		{
			"members": [["alice@alice.com"],["bob@gmail.com"],[]]
		}
	`)

	// chaining plucks should be equivilent to nesting
	alice.Query(`members().pluck(email_addresses).pluck(addr)`).ShouldReturn(`
		{
			"members": [["alice@alice.com"],["bob@gmail.com"],[]]
		}
	`)

	// first() should work on the result of a pluck
	alice.Query(`members().pluck(email_addresses).first(){addr}`).ShouldReturn(`
		{
			"members": [{"addr":"alice@alice.com"}]
		}
	`)

	// should be possible to alias complex plucks
	alice.Query(`first_email: members().pluck(email_addresses).pluck(addr).first()`).ShouldReturn(`
		{
			"first_email": ["alice@alice.com"]
		}
	`)

	// count() should work on plucks
	alice.Query(`members().pluck(email_addresses).pluck(addr).count()`).ShouldReturn(`
		{
			"members": 3
		}
	`)

	// cannot pluck on non-arrays
	alice.Query(`members().pluck(email_addresses).pluck(addr).pluck(addr)`).ShouldFail()
	alice.Query(`members().pluck(email_addresses).pluck(addr.first())`).ShouldFail()

	alice.Query(`members().pluck(email_addresses.pluck(addr).first())`).ShouldReturn(`
		{
			"members": ["alice@alice.com","bob@gmail.com", null]
		}
	`)

	// take should work on plucked results
	alice.Query(`members().pluck(email_addresses.pluck(addr).first()).take(1)`).ShouldReturn(`
		{
			"members": ["alice@alice.com"]
		}
	`)

	// ----------------------------------------

	// test relationships
	alice.Exec("addFriend", bob.ID.String()).ShouldSucceed()
	bob.Exec("addFriend", alice.ID.String()).ShouldFail() // already friends
	kate.Exec("addFriend", alice.ID.String()).ShouldSucceed()
	alice.Query(`me(){friends().pluck(username)}`).ShouldReturn(`
		{
			"me":{
				"friends": ["bob","kate"]
			}
		}
	`)
	bob.Query(`me(){friends().pluck(username)}`).ShouldReturn(`
		{
			"me":{
				"friends": ["alice"]
			}
		}
	`)

	// should be possible to nest relationship queries indefinitily
	alice.Query(`
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
	`).ShouldReturn(`
		{
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
		}
	`)

	// friends shouldnt be able to see friends passwords as it's not in the select
	alice.Query(`
		me(){
			friends() {
				password
			}
		}
	`).ShouldFail()

	// first() should return first entry
	alice.Query(`
		me(){
			friends().first() {
				username
			}
		}
	`).ShouldReturn(`
		{
			"me":{
				"friends": {"username":"bob"}
			}
		}
	`)

	// -------------------------------------

	// test global CTE
	//
	// the call to shaowed_members() should only return members with
	// username.length < 4 ... even tho the SQL just says "select * from shadowed_members"
	// this is because the root() call declares a `with` CTE that is inserted into the query
	alice.Query(`
		members().first(){
			shadowed_members(){
				username
			}
		}

	`).ShouldReturn(`
		{
			"members": {
				"shadowed_members": [
					{"username":"bob"}
				]
			}
		}
	`)

	// ----------------------------------------

	// use take() filter to grab just the first member
	alice.Query(`members().take(1){username}`).ShouldReturn(`
		{
			"members": [
				{"username":"alice"}
			]
		}
	`)

	// use first() filter to grab just the first member AND just grab the object
	alice.Query(`members().first(){username}`).ShouldReturn(`
		{
			"members": {"username":"alice"}
		}
	`)

	// use slice(0,2) to perform an OFFSET=2 LIMIT=1
	alice.Query(`
		members().slice(2,1) {
			username
		}
	`).ShouldReturn(`
		{
			"members": [
				{"username":"kate"}
			]
		}
	`)

	// filter should allow simple where-style clauses
	alice.Query(`
		me(){
			friends.filter(id = "` + bob.ID.String() + `") {
				username
			}
		}
	`).ShouldReturn(`
		{
			"me":{
				"friends": [{"username":"bob"}]
			}
		}
	`)

	// aliases should allow same properties to be reused
	// placeholders can be used for arguments
	alice.Query(`
		me(){
			bob:friends.filter(id = $1).first() {
				username
			}
			kate:friends.filter($2 = id).first() {
				username
			}
		}
	`, bob.ID.String(), kate.ID.String()).ShouldReturn(`
		{
			"me":{
				"bob": {"username":"bob"},
				"kate": {"username":"kate"}
			}
		}
	`)

	// should be possible to pass in placeholder args to query calls
	alice.Query(`
		me(){
			member_by_id($1) {
				username
			}
		}
	`, bob.ID.String()).ShouldReturn(`
		{
			"me":{
				"member_by_id": {"username":"bob"}
			}
		}
	`)

	// -------------------

	// should be possible to use arbitary SQL queries for results
	alice.Query(`numbers`).ShouldReturn(`
		{
			"numbers":[10,5,11]
		}
	`)

	// ..and sort them
	alice.Query(`numbers.sort()`).ShouldReturn(`
		{
			"numbers":[5,10,11]
		}
	`)

	// should be possible to sort on a specific property
	alice.Query(`members().sort(username).pluck(username)`).ShouldReturn(`
		{
			"members": ["alice","bob","kate"]
		}
	`)

	// ...and set the direction
	alice.Query(`members().sort(username desc).pluck(username)`).ShouldReturn(`
		{
			"members": ["kate","bob","alice"]
		}
	`)

	// should be possible to sort on simple arrays with a direction
	alice.Query(`members().pluck(username).sort(desc)`).ShouldReturn(`
		{
			"members": ["kate","bob","alice"]
		}
	`)

	// -------------------

	// identical simple properties should be merged
	bob.Query(`
		members().first() {
			username
			username
		}
	`).ShouldReturn(`
		{
			"members": {"username":"alice"}
		}
	`)

	// identical dynamic properties should be merged
	bob.Query(`
		members().first(){
			id
		}
		members().first(){
			username
		}
	`).ShouldReturn(`
		{
			"members": {"username":"alice","id":"` + alice.ID.String() + `"}
		}
	`)

	// identical dynamic filter properties can be merged
	bob.Query(`
		members().filter(username = "bob"){
			id
		}
		members().filter("bob" = username){
			username
		}
	`).ShouldReturn(`
		{
			"members": [{"username":"bob","id":"` + bob.ID.String() + `"}]
		}
	`)

	// identical complex properties should be merged
	bob.Query(`
		members(){
			email_addresses.pluck(addr)
		}
		members(){
			username
		}
	`).ShouldReturn(`
		{
			"members": [
				{"username":"alice", "email_addresses":["alice@alice.com"]},
				{"username":"bob", "email_addresses":["bob@gmail.com"]},
				{"username":"kate", "email_addresses":[]}
			]
		}
	`)

	// non-identical dynamic properties cannot be merged
	bob.Query(`
		members().first(){
			id
		}
		members(){
			username
		}
	`).ShouldFail()

	// clashing aliases cannot be merged
	bob.Query(`
		people:members().take(1){username}
		people:members().take(2){username}
	`).ShouldFail()

	// -----------------------------

	// Country data should be populated from bootstrap statement
	bob.Query(`
		countries(){
			name
			code
		}
	`).ShouldReturn(`
		{
			"countries": [
				{"name":"United Kingdom", "code":"GB"},
				{"name":"France", "code":"FR"}
			]
		}
	`)

	// -----------------------------

	// Someflag should be set in authenticate and stored in the session token
	bob.Query(`
		someflag
	`).ShouldReturn(`
		{
			"someflag": true
		}
	`)

	// -----------------------------

	// alice should be indestructable
	alice.Exec("destroyMember").ShouldFail()

	// ..but bob is not
	bob.Exec("destroyMember").ShouldSucceed()

	// check that cascading deletes removed most of the email addrs
	alice.Query(`remaining: email_addresses.count()`).ShouldReturn(`
		{"remaining":1}
	`)

	// execute the tests!
	for _, tc := range tests {
		if err := tc.Test(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestInfo(t *testing.T) {
	res, err := http.Get("http://localhost/info")
	if err != nil {
		t.Fatal(err)
	}
	info := &schema.Info{}
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &info); err != nil {
		t.Fatal(err)
	}
	if info.Version != 2 {
		t.Fatal("expected version=1")
	}
	found := false
	for _, name := range info.Mutations {
		if name == "registerMember" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected registerMember to appear in the list of info.Mutations")
	}
}

func TestMain(m *testing.M) {
	// create a tmp dir
	tmp, err := ioutil.TempDir("", "arlatestdata")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmp)
	// write a mutation log
	f, err := os.Create(filepath.Join(tmp, "datastore"))
	if err != nil {
		log.Fatal(err)
	}
	// Add an entry for a mutation with an older version number
	// so we can test transforming
	err = json.NewEncoder(f).Encode(&schema.Mutation{
		Name:    "createUser",
		Args:    []interface{}{alice.ID, alice.Username, alice.Password},
		Version: 1,
	})
	f.Close()
	if err != nil {
		log.Fatal(err)
	}
	// start server
	server := New(Config{
		ConfigPath:     "config.js",
		DataDir:        tmp,
		Secret:         "mysecret",
		Debug:          true,
		MaxConnections: 5,
	})
	if err := server.Start(); err != nil {
		log.Fatal("failed to start server", err)
	}
	// FIXME: there's a race condition between calling Start() and the HTTP server actually
	// accepting connections ... this only really affects the tests tho.
	time.Sleep(1 * time.Second)
	// run tests
	status := m.Run()
	defer os.Exit(status)
	if status == 0 {
		fmt.Println(ansi.Green, "PASS", ansi.Reset)
	} else {
		fmt.Println(ansi.Red, "TESTS FAILED", ansi.Reset)
	}
	// shutdown server
	if err := server.Stop(); err != nil {
		fmt.Println("failed to cleanly stop server:", err)
	}
}
