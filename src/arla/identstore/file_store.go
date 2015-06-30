package ident

import (
	"encoding/gob"
	"os"
	"sync"
)

// A simple file store that encodes the data in gob
type FileStore struct {
	Path string
	mu   sync.Mutex
	*MemoryStore
}

// Stores idents in a gob encoded file at the given path.
func NewFileStore(path string) (*FileStore, error) {
	fs := &FileStore{}
	fs.Path = path
	fs.MemoryStore = NewMemoryStore()
	return fs, fs.load()
}

func (fs *FileStore) Put(id *Identity) (err error) {
	err = fs.MemoryStore.Put(id)
	if err != nil {
		return
	}
	return fs.save()
}

// Save idents as gob
func (fs *FileStore) save() (err error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	f, err := os.Create(fs.Path)
	if err != nil {
		return
	}
	defer f.Close()
	dec := gob.NewEncoder(f)
	fs.MemoryStore.mu.Lock()
	defer fs.MemoryStore.mu.Unlock()
	err = dec.Encode(fs.MemoryStore.ids)
	if err != nil {
		return
	}
	return
}

// Fetch gob
func (fs *FileStore) load() (err error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.MemoryStore = NewMemoryStore()
	if _, err := os.Stat(fs.Path); err == nil {
		f, err := os.Open(fs.Path)
		if err != nil {
			return err
		}
		defer f.Close()
		dec := gob.NewDecoder(f)
		ids := make(map[UUID]*Identity)
		err = dec.Decode(&ids)
		if err != nil {
			return err
		}
		fs.MemoryStore.load(ids)
	}
	return
}
