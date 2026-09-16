// Package usecase holds the application business rules: one type per
// use case, each depending only on the domain layer and on the
// ItemRepository port defined here. Concrete adapters (SQLite, in-memory,
// ...) implement this port; use cases never know which one is plugged in.
package usecase

import (
	"context"

	"github.com/gustavogmartinelli/gtd/internal/domain"
)

// ListFilter narrows a FindAll query. A zero value means "all statuses,
// non-completed only".
type ListFilter struct {
	Status           domain.Status
	IncludeCompleted bool
}

// ItemRepository is the port through which use cases persist and retrieve
// items. It is defined by its consumers (the use cases), not by its
// implementers — the Dependency Inversion Principle at the architecture's
// boundary.
type ItemRepository interface {
	Save(ctx context.Context, item *domain.Item) error
	FindByID(ctx context.Context, id int64) (*domain.Item, error)
	FindAll(ctx context.Context, filter ListFilter) ([]*domain.Item, error)
	Update(ctx context.Context, item *domain.Item) error
	Delete(ctx context.Context, id int64) error
}
