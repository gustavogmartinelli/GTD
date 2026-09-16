package model

import "time"

// Status values an Item can hold in the GTD workflow.
const (
	StatusInbox = "inbox"
	StatusNext  = "next"
)

type Item struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Notes       string     `json:"notes,omitempty"`
	Status      string     `json:"status"`
	Context     string     `json:"context,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}
