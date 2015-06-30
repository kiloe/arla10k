package querystore

import (
	"arla/schema"

	"code.google.com/p/go-uuid/uuid"
)

// Engine interface defines the methods required by a query store
type Engine interface {
	Start() error
	Stop() error
	Mutate(*schema.Mutation) error
	Query(id uuid.UUID, q string) ([]byte, error)
}

// Config defines options configuring the query engine
type Config struct {
	Path string
}

// New creates a new query engine (which is always postgres at the moment)
func New(cfg *Config) (e Engine) {
	return &postgres{cfg: cfg}
}
