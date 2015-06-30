package querystore

import (
	"arla/schema"
	"io"
)

// Engine interface defines the methods required by a query store
type Engine interface {
	Start() error
	Stop() error
	Wait() error
	Mutate(*schema.Mutation) error
	Query(id schema.UUID, q string, w io.Writer) error
}

// Config defines options configuring the query engine
type Config struct {
	Path string
}

// New creates a new query engine (which is always postgres at the moment)
func New(cfg *Config) (e Engine) {
	return &postgres{cfg: cfg}
}
