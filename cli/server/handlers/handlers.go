package handlers

import (
	"net/http"

	"github.com/go-chi/chi"
)

// Handler returns an http.Handler that exposes the service resources.
func Handler() http.Handler {
	r := chi.NewRouter()
	// Setup stage endpoint
	r.Mount("/setup", func() http.Handler {
		sr := chi.NewRouter()
		sr.Post("/", HandleLiveness())
		return sr
	}())
	return r
}
