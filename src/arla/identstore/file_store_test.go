package ident

import (
	"io/ioutil"
	"testing"
)

func dummyFileStore(t *testing.T) (*Set, *FileStore) {
	s, err := ioutil.TempDir("/tmp", "ident")
	if err != nil {
		t.Fatal(err)
	}
	fs, err := NewFileStore(s + "/ident.fs")
	if err != nil {
		t.Fatal(err)
	}
	set, err := NewSet(fs)
	if err != nil {
		t.Fatal(err)
	}
	dummyData(t, set)
	return set, fs
}

func TestFileStoreReload(t *testing.T) {
	s, fs := dummyFileStore(t)
	err := fs.load()
	if err != nil {
		t.Fatal(err)
	}
	u, err := s.FindByUsername("user-1")
	if err != nil {
		t.Fatal(err)
	}
	if u.Username != "user-1" {
		t.Errorf("Expected reloaded Username to be user-1 got %s", u.Username)
	}
}
