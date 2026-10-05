package usecase

import (
	"context"

	"github.com/gustavogmartinelli/gtd/internal/domain"
)

type ListItemsInput struct {
	Status           domain.Status
	IncludeCompleted bool
}

// ListItemsUseCase returns items, optionally filtered by status.
type ListItemsUseCase struct {
	repo ItemRepository
}

func NewListItemsUseCase(repo ItemRepository) *ListItemsUseCase {
	return &ListItemsUseCase{repo: repo}
}

func (uc *ListItemsUseCase) Execute(ctx context.Context, in ListItemsInput) ([]*domain.Item, error) {
	if in.Status != "" && !in.Status.Valid() {
		return nil, domain.ErrInvalidStatus
	}
	return uc.repo.FindAll(ctx, ListFilter{
		Status:           in.Status,
		IncludeCompleted: in.IncludeCompleted,
	})
}
