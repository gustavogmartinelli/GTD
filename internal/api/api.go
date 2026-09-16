// Package api exposes the GTD store over an HTTP/JSON API.
package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gustavogmartinelli/gtd/internal/model"
	"github.com/gustavogmartinelli/gtd/internal/store"
)

type API struct {
	store *store.Store
}

func New(s *store.Store) *API {
	return &API{store: s}
}

// Router builds the HTTP handler with all routes registered.
func (a *API) Router() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", a.handleHealth)

	mux.HandleFunc("POST /items", a.handleCapture)
	mux.HandleFunc("GET /items", a.handleList)
	mux.HandleFunc("GET /items/{id}", a.handleGet)
	mux.HandleFunc("PATCH /items/{id}", a.handleUpdate)
	mux.HandleFunc("DELETE /items/{id}", a.handleDelete)
	mux.HandleFunc("POST /items/{id}/process", a.handleProcess)
	mux.HandleFunc("POST /items/{id}/complete", a.handleComplete)

	return logging(mux)
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		log.Printf("%s %s", r.Method, r.URL.Path)
	})
}

func (a *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type captureRequest struct {
	Title string `json:"title"`
	Notes string `json:"notes"`
}

// handleCapture adds a new item to the inbox. This is the GTD "capture"
// step: get it out of your head with zero friction, process it later.
func (a *API) handleCapture(w http.ResponseWriter, r *http.Request) {
	var req captureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	item, err := a.store.Capture(r.Context(), req.Title, req.Notes)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

// handleList returns items. Query params:
//   - status: "inbox" or "next" (omit for all)
//   - include_completed: "true" to include completed items
func (a *API) handleList(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status != "" && status != model.StatusInbox && status != model.StatusNext {
		writeError(w, http.StatusBadRequest, "status must be 'inbox' or 'next'")
		return
	}
	includeCompleted := r.URL.Query().Get("include_completed") == "true"

	items, err := a.store.List(r.Context(), status, includeCompleted)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []*model.Item{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *API) handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := a.store.Get(r.Context(), id)
	if handleStoreErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, item)
}

type updateRequest struct {
	Title   *string `json:"title"`
	Notes   *string `json:"notes"`
	Status  *string `json:"status"`
	Context *string `json:"context"`
}

func (a *API) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Status != nil && *req.Status != model.StatusInbox && *req.Status != model.StatusNext {
		writeError(w, http.StatusBadRequest, "status must be 'inbox' or 'next'")
		return
	}

	item, err := a.store.Update(r.Context(), id, req.Title, req.Notes, req.Status, req.Context)
	if handleStoreErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	err = a.store.Delete(r.Context(), id)
	if handleStoreErr(w, err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type processRequest struct {
	Context string `json:"context"`
	Notes   string `json:"notes"`
}

// handleProcess implements the GTD "clarify" step for a single inbox
// item: decide it's actionable and turn it into a next action, tagged
// with the context needed to do it (e.g. "@home", "@calls").
func (a *API) handleProcess(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req processRequest
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
	}

	existing, err := a.store.Get(r.Context(), id)
	if handleStoreErr(w, err) {
		return
	}
	notes := existing.Notes
	if req.Notes != "" {
		notes = req.Notes
	}

	item, err := a.store.ProcessToNext(r.Context(), id, req.Context, notes)
	if handleStoreErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) handleComplete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := a.store.Complete(r.Context(), id)
	if handleStoreErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

func handleStoreErr(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "item not found")
		return true
	}
	writeError(w, http.StatusInternalServerError, err.Error())
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
