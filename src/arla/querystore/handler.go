package querystore

import "net/http"

// Handler is an HTTP handler for accessing the querystore
type Handler struct {
	Engine Engine
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

}

// NewHandler creates a new Handler for the given engine Config
func NewHandler(qs Engine) http.Handler {
	return &Handler{Engine: qs}
}
