package ident

import (
	"strconv"
	"testing"
	"time"
)

const (
	pass = "x1x2x3x4"
)

var testSet *Set

func dummyData(t *testing.T, s *Set) {
	for i := 0; i < 2; i++ {
		u, err := NewIdentity("user-"+strconv.Itoa(i), pass)
		if err != nil {
			t.Fatal(err)
		}
		err = s.Add(u)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func dummySet(t *testing.T) *Set {
	var err error
	if testSet == nil {
		testSet, err = NewSet(NewMemoryStore())
		if err != nil {
			t.Fatal(err)
		}
		dummyData(t, testSet)
	}
	return testSet
}

func TestLogin(t *testing.T) {
	s := dummySet(t)
	u, _, err := s.Login("user-0", pass)
	if err != nil {
		t.Fatal(err)
	}
	if u.Username != "user-0" {
		t.Errorf("Expected returned Username to be %s got %s", "user-0", u.Username)
	}
}

func TestToken(t *testing.T) {
	s := dummySet(t)
	u1, tok1, err := s.Login("user-1", pass)
	if err != nil {
		t.Fatal(err)
	}
	_, tok2, err := s.Login("user-1", pass)
	if err != nil {
		t.Fatal(err)
	}
	u2, err := s.FindByToken(tok1)
	if err != nil {
		t.Fatal("tok1:", err)
	}
	if u1 != u2 {
		t.Fatal("Expected tok1 to return same ident as Login")
	}
	u3, err := s.FindByToken(tok2)
	if err != nil {
		t.Fatal("tok2:", err)
	}
	if u1 != u3 {
		t.Fatal("Expected tok2 to return same ident as Login")
	}
}

func TestTokenExpirey(t *testing.T) {
	s := dummySet(t)
	s.tokenDuration = "0.1s"
	_, tok, err := s.Login("user-1", pass)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1 * time.Second)
	_, err = s.FindByToken(tok)
	if err == nil {
		t.Fatal("Expected token to have expired")
		t.Fatal(err)
	}
}

func TestLogout(t *testing.T) {
	s := dummySet(t)
	_, tok, err := s.Login("user-1", pass)
	if err != nil {
		t.Error(err)
		return
	}
	err = s.Logout(tok, false)
	if err != nil {
		t.Error(err)
		return
	}
	_, err = s.FindByToken(tok)
	if err == nil {
		t.Fatal("Expected token to have been revoked")
	}
}

func TestLogoutAll(t *testing.T) {
	s := dummySet(t)
	s.tokenDuration = "1s"
	_, tok1, err := s.Login("user-1", pass)
	if err != nil {
		t.Error(err)
		return
	}
	_, tok2, err := s.Login("user-1", pass)
	if err != nil {
		t.Error(err)
		return
	}
	err = s.Logout(tok2, true)
	if err != nil {
		t.Error(err)
		return
	}
	_, err = s.FindByToken(tok1)
	if err == nil {
		t.Fatal("Expected tok1 to have been revoked")
	}
	_, err = s.FindByToken(tok2)
	if err == nil {
		t.Fatal("Expected tok1 to have been revoked")
	}
}
