package rest

import (
	"log"
	"net/http"
)

// NewRouter wires HTTP routes to the handler's controller methods.
func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", h.handleHealth)

	mux.HandleFunc("POST /items", h.handleCapture)
	mux.HandleFunc("GET /items", h.handleList)
	mux.HandleFunc("GET /items/{id}", h.handleGet)
	mux.HandleFunc("PATCH /items/{id}", h.handleUpdate)
	mux.HandleFunc("DELETE /items/{id}", h.handleDelete)
	mux.HandleFunc("POST /items/{id}/process", h.handleProcess)
	mux.HandleFunc("POST /items/{id}/complete", h.handleComplete)

	return logging(mux)
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		log.Printf("%s %s", r.Method, r.URL.Path)
	})
}
