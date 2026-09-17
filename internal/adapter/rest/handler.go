// Package rest is an "interface adapters" component: it translates HTTP
// requests into use case input, and use case output into HTTP responses.
// It depends on the usecase and domain layers, never the other way
// around, and knows nothing about SQLite or any other storage detail.
package rest

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gustavogmartinelli/gtd/internal/domain"
	"github.com/gustavogmartinelli/gtd/internal/usecase"
)

// Handler holds one use case per GTD operation. Each is referenced only
// through its own small struct type, so this adapter depends on exactly
// the application behavior it needs.
type Handler struct {
	capture  *usecase.CaptureItemUseCase
	list     *usecase.ListItemsUseCase
	get      *usecase.GetItemUseCase
	process  *usecase.ProcessItemUseCase
	update   *usecase.UpdateItemUseCase
	complete *usecase.CompleteItemUseCase
	delete   *usecase.DeleteItemUseCase
}

func NewHandler(
	capture *usecase.CaptureItemUseCase,
	list *usecase.ListItemsUseCase,
	get *usecase.GetItemUseCase,
	process *usecase.ProcessItemUseCase,
	update *usecase.UpdateItemUseCase,
	complete *usecase.CompleteItemUseCase,
	delete *usecase.DeleteItemUseCase,
) *Handler {
	return &Handler{
		capture:  capture,
		list:     list,
		get:      get,
		process:  process,
		update:   update,
		complete: complete,
		delete:   delete,
	}
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleCapture(w http.ResponseWriter, r *http.Request) {
	var req captureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	item, err := h.capture.Execute(r.Context(), usecase.CaptureItemInput{
		Title: req.Title,
		Notes: req.Notes,
	})
	if writeIfErr(w, err) {
		return
	}
	writeJSON(w, http.StatusCreated, newItemResponse(item))
}

func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	status := domain.Status(r.URL.Query().Get("status"))
	includeCompleted := r.URL.Query().Get("include_completed") == "true"

	items, err := h.list.Execute(r.Context(), usecase.ListItemsInput{
		Status:           status,
		IncludeCompleted: includeCompleted,
	})
	if writeIfErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, newItemListResponse(items))
}

func (h *Handler) handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.get.Execute(r.Context(), id)
	if writeIfErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, newItemResponse(item))
}

func (h *Handler) handleUpdate(w http.ResponseWriter, r *http.Request) {
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

	var status *domain.Status
	if req.Status != nil {
		s := domain.Status(*req.Status)
		status = &s
	}

	item, err := h.update.Execute(r.Context(), usecase.UpdateItemInput{
		ID:      id,
		Title:   req.Title,
		Notes:   req.Notes,
		Status:  status,
		Context: req.Context,
	})
	if writeIfErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, newItemResponse(item))
}

func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.delete.Execute(r.Context(), id); writeIfErr(w, err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleProcess(w http.ResponseWriter, r *http.Request) {
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

	item, err := h.process.Execute(r.Context(), usecase.ProcessItemInput{
		ID:      id,
		Context: req.Context,
		Notes:   req.Notes,
	})
	if writeIfErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, newItemResponse(item))
}

func (h *Handler) handleComplete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.complete.Execute(r.Context(), id)
	if writeIfErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, newItemResponse(item))
}

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

// writeIfErr maps domain/use-case errors onto HTTP status codes and
// writes the error response. It returns true if an error was written, so
// callers can `if writeIfErr(w, err) { return }`.
func writeIfErr(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrEmptyTitle),
		errors.Is(err, domain.ErrInvalidStatus),
		errors.Is(err, domain.ErrAlreadyCompleted):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		log.Printf("internal error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
	}
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
