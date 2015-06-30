package identstore

import (
	"arla/schema"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var tmpdir string
var user1 *schema.User
var user2 *schema.User

func TestMain(m *testing.M) {
	// state dir
	var err error
	tmpdir, err = ioutil.TempDir("", "waltest")
	if err != nil {
		log.Fatal(err)
	}
	// user1
	user1 = &schema.User{
		Name:    "admin",
		ID:      schema.TimeUUID(),
		Aliases: []string{"admin", "administrator", "su", "superuser", "god"},
	}
	if err := user1.SetPassword("test1"); err != nil {
		log.Fatal(err)
	}
	// user2
	user2 = &schema.User{
		Name:    "jeff",
		ID:      schema.MustRandomUUID(),
		Aliases: []string{"jeff", "jeff@jeff.com"},
	}
	if err := user1.SetPassword("test1"); err != nil {
		log.Fatal(err)
	}
	// run
	status := m.Run()
	// cleanup
	if err = os.RemoveAll(tmpdir); err != nil {
		log.Fatal(err)
	}
	os.Exit(status)
}

func TestGetById(t *testing.T) {
	filename := filepath.Join(tmpdir, "TestGetById")
	s, err := Open(filename)
	if err != nil {
		t.Fatal("open ", err)
	}
	// write users
	if err := s.Put(user1); err != nil {
		t.Fatal("put ", err)
	}
	if err := s.Put(user2); err != nil {
		t.Fatal("put ", err)
	}
	// get user1 by id
	res := s.Get(user1.ID)
	if res == nil {
		t.Fatal("expected to fetch user1 from store using ID got nil")
	}
	if res.ID != user1.ID {
		t.Fatal("expected to fetch user1 from store using ID got incorrect ID")
	}
	// get user2 by id
	res = s.Get(user2.ID)
	if res == nil {
		t.Fatal("expected to fetch user2 from store using ID got nil")
	}
	if res.ID != user2.ID {
		t.Fatal("expected to fetch user2 from store using ID got incorrect ID")
	}
}

func TestGetByAlias(t *testing.T) {
	filename := filepath.Join(tmpdir, "TestGetByAlias")
	s, err := Open(filename)
	if err != nil {
		t.Fatal("open ", err)
	}
	// write user
	if err := s.Put(user1); err != nil {
		t.Fatal("put ", err)
	}
	if err := s.Put(user2); err != nil {
		t.Fatal("put ", err)
	}
	// fetch user1 by alias
	res := s.Find("administrator")
	if res == nil {
		t.Fatal("expected to fetch user1 from store using alias got nil")
	}
	if res.ID != user1.ID {
		t.Fatal("expected to fetch user1 from store using alias got incorrect ID")
	}
	// fetch user2 by alias
	res = s.Find("jeff@jeff.com")
	if res == nil {
		t.Fatal("expected to fetch user2 from store using alias got nil")
	}
	if res.ID != user2.ID {
		t.Fatal("expected to fetch user2 from store using alias got incorrect ID")
	}
}

func TestReopen(t *testing.T) {
	filename := filepath.Join(tmpdir, "TestReopen")
	s, err := Open(filename)
	if err != nil {
		t.Fatal("open ", err)
	}
	// write user
	if err := s.Put(user1); err != nil {
		t.Fatal("put ", err)
	}
	if err := s.Put(user2); err != nil {
		t.Fatal("put ", err)
	}
	// reopen
	s, err = Open(filename)
	if err != nil {
		t.Fatal("open ", err)
	}
	// fetch user1 by alias
	res := s.Find("administrator")
	if res == nil {
		t.Fatal("expected to fetch user1 from store using alias got nil")
	}
	if res.ID != user1.ID {
		t.Fatal("expected to fetch user1 from store using alias got incorrect ID")
	}
	// fetch user2 by alias
	res = s.Find("jeff@jeff.com")
	if res == nil {
		t.Fatal("expected to fetch user2 from store using alias got nil")
	}
	if res.ID != user2.ID {
		t.Fatal("expected to fetch user2 from store using alias got incorrect ID")
	}
}

func TestUpdateUser(t *testing.T) {
	filename := filepath.Join(tmpdir, "TestUpdateUser")
	s, err := Open(filename)
	if err != nil {
		t.Fatal("open ", err)
	}
	// write user
	if err := s.Put(user1); err != nil {
		t.Fatal("put ", err)
	}
	newman := &schema.User{
		ID:   user1.ID,
		Name: "newman",
	}
	if err := s.Put(newman); err != nil {
		t.Fatal("put ", err)
	}
	// fetch newman by old alias
	res := s.Find("admin")
	if res != nil {
		t.Fatal("expected to fetching newman by he's old alias to fail but it did not")
	}
}

func TestPutInvalidId(t *testing.T) {
	filename := filepath.Join(tmpdir, "TestPutInvalidId")
	s, err := Open(filename)
	if err != nil {
		t.Fatal("open ", err)
	}
	// write user
	id, _ := schema.ParseUUID("")
	newman := &schema.User{
		ID:   id,
		Name: "newman",
	}
	if err := s.Put(newman); err == nil {
		t.Fatal("expected adding a user without an ID to fail but it passed")
	}
}

func TestPassword(t *testing.T) {
	filename := filepath.Join(tmpdir, "TestPassword")
	s, err := Open(filename)
	if err != nil {
		t.Fatal("open ", err)
	}
	// write user
	if err := s.Put(user1); err != nil {
		t.Fatal("put ", err)
	}
	// fetch by id
	res := s.Get(user1.ID)
	// check password still matches
	if !res.MatchPassword("test1") {
		t.Fatal("expected user1 password to match")
	}
	// check password is not stored as plain text!
	if strings.Contains(string(res.Hash), "test1") {
		t.Fatal("expected user1 password not to be stored as plain text")
	}
	// update pass
	if err := res.SetPassword("newpass"); err != nil {
		t.Fatal(err)
	}
	// By design ?...
	//if res2 := s.Get(string(user1.ID)); res2.MatchPassword("newpass") {
	//	t.Fatal("did not expect password to be updated without Put being called!")
	//}
	// write back
	if err := s.Put(res); err != nil {
		t.Fatal("put ", err)
	}
	// now it should be updated
	if res2 := s.Get(user1.ID); !res2.MatchPassword("newpass") {
		t.Fatal("expected password to be updated after")
	}

}

func TestClashingAlias(t *testing.T) {
	filename := filepath.Join(tmpdir, "TestClashingAlias")
	s, err := Open(filename)
	if err != nil {
		t.Fatal("open ", err)
	}
	alias := "monkeychops"
	// write two users with unique ids but same alias
	a := &schema.User{
		ID:      schema.MustRandomUUID(),
		Aliases: []string{alias},
	}
	b := &schema.User{
		ID:      schema.MustRandomUUID(),
		Aliases: []string{alias},
	}
	if err := s.Put(a); err != nil {
		t.Fatal(err)
	}
	if err := s.Put(b); err == nil {
		t.Fatal("expected adding a user with clashing alias to fail")
	}
}
