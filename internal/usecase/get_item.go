package usecase

import (
	"context"

	"github.com/gustavogmartinelli/gtd/internal/domain"
)

// GetItemUseCase fetches a single item by id.
type GetItemUseCase struct {
	repo ItemRepository
}

func NewGetItemUseCase(repo ItemRepository) *GetItemUseCase {
	return &GetItemUseCase{repo: repo}
}

func (uc *GetItemUseCase) Execute(ctx context.Context, id int64) (*domain.Item, error) {
	return uc.repo.FindByID(ctx, id)
}
