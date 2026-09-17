package rest

import (
	"time"

	"github.com/gustavogmartinelli/gtd/internal/domain"
)

// itemResponse is the outbound presentation of a domain.Item. Keeping it
// separate from the entity means JSON shape and tags are a REST-adapter
// concern, not something the domain layer needs to know about.
type itemResponse struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Notes       string     `json:"notes,omitempty"`
	Status      string     `json:"status"`
	Context     string     `json:"context,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

func newItemResponse(item *domain.Item) itemResponse {
	return itemResponse{
		ID:          item.ID,
		Title:       item.Title,
		Notes:       item.Notes,
		Status:      string(item.Status),
		Context:     item.Context,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
		CompletedAt: item.CompletedAt,
	}
}

func newItemListResponse(items []*domain.Item) []itemResponse {
	res := make([]itemResponse, len(items))
	for i, item := range items {
		res[i] = newItemResponse(item)
	}
	return res
}

type captureRequest struct {
	Title string `json:"title"`
	Notes string `json:"notes"`
}

type updateRequest struct {
	Title   *string `json:"title"`
	Notes   *string `json:"notes"`
	Status  *string `json:"status"`
	Context *string `json:"context"`
}

type processRequest struct {
	Context string `json:"context"`
	Notes   string `json:"notes"`
}
