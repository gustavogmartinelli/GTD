// Package domain holds the enterprise business rules: the Item entity and
// the invariants it must uphold. It has no dependency on any other layer —
// not the use cases, not the database, not HTTP.
package domain

import (
	"errors"
	"strings"
	"time"
)

// Status is the position of an Item in the GTD workflow.
type Status string

const (
	StatusInbox Status = "inbox"
	StatusNext  Status = "next"
)

func (s Status) Valid() bool {
	return s == StatusInbox || s == StatusNext
}

var (
	ErrEmptyTitle       = errors.New("title is required")
	ErrInvalidStatus    = errors.New("status must be 'inbox' or 'next'")
	ErrAlreadyCompleted = errors.New("item is already completed")
	ErrNotFound         = errors.New("item not found")
)

// Item is a single piece of "stuff": something captured to be clarified
// and, eventually, acted on.
type Item struct {
	ID          int64
	Title       string
	Notes       string
	Status      Status
	Context     string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time
}

// NewItem constructs a new inbox item, enforcing the entity's invariants.
// This is the GTD "capture" step.
func NewItem(title, notes string) (*Item, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, ErrEmptyTitle
	}
	now := time.Now().UTC()
	return &Item{
		Title:     title,
		Notes:     notes,
		Status:    StatusInbox,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// Process is the GTD "clarify" step: decide the item is actionable, turn
// it into a next action, and tag it with the context needed to do it.
func (i *Item) Process(gtdContext, notes string) {
	i.Status = StatusNext
	i.Context = gtdContext
	if notes != "" {
		i.Notes = notes
	}
	i.touch()
}

// Rename changes the item's title, enforcing the same invariant as NewItem.
func (i *Item) Rename(title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return ErrEmptyTitle
	}
	i.Title = title
	i.touch()
	return nil
}

func (i *Item) SetNotes(notes string) {
	i.Notes = notes
	i.touch()
}

func (i *Item) SetStatus(status Status) error {
	if !status.Valid() {
		return ErrInvalidStatus
	}
	i.Status = status
	i.touch()
	return nil
}

func (i *Item) SetContext(gtdContext string) {
	i.Context = gtdContext
	i.touch()
}

// Complete marks the item done. An already-completed item cannot be
// completed again.
func (i *Item) Complete() error {
	if i.IsCompleted() {
		return ErrAlreadyCompleted
	}
	now := time.Now().UTC()
	i.CompletedAt = &now
	i.touch()
	return nil
}

func (i *Item) IsCompleted() bool {
	return i.CompletedAt != nil
}

func (i *Item) touch() {
	i.UpdatedAt = time.Now().UTC()
}
