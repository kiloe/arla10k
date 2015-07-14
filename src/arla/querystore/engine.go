package querystore

import (
	"arla/schema"
	"io"
	"os"
)

// Engine interface defines the methods required by a query store
type Engine interface {
	Start() error
	Stop() error
	Wait() error
	Mutate(*schema.Mutation) error
	Query(*schema.Query, io.Writer) error
	NewWriter() (w io.WriteCloser, err error)
	SetLogLevel(logLevel)
	GetLogLevel() logLevel
	Authenticate(string) (schema.Token, error)
	Register(string) (*schema.Mutation, error)
}

// Config defines options configuring the query engine
type Config struct {
	Path     string
	LogLevel logLevel
}

// New creates a new query engine (which is always postgres at the moment)
func New(cfg *Config) (e Engine, err error) {
	p := &postgres{
		cfg: cfg,
		log: NewLogFormatter(os.Stderr),
	}
	p.SetLogLevel(cfg.LogLevel)
	return p, p.Start()
}
