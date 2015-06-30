package ident

// Save to a persistant store
type Putter interface {
	Put(*Identity) error
}

// Remove from a persistant store
type Deleter interface {
	Delete(*Identity) error
}

// Get from a persistant store
type Finder interface {
	FindById(UUID) (*Identity, error)
	FindByUsername(string) (*Identity, error)
}

// The interface for different identity stores
type Storer interface {
	Finder
	Putter
	Deleter
}
