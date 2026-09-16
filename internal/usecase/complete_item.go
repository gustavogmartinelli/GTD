package usecase

import (
	"context"

	"github.com/gustavogmartinelli/gtd/internal/domain"
)

// CompleteItemUseCase marks an item done.
type CompleteItemUseCase struct {
	repo ItemRepository
}

func NewCompleteItemUseCase(repo ItemRepository) *CompleteItemUseCase {
	return &CompleteItemUseCase{repo: repo}
}

func (uc *CompleteItemUseCase) Execute(ctx context.Context, id int64) (*domain.Item, error) {
	item, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := item.Complete(); err != nil {
		return nil, err
	}
	if err := uc.repo.Update(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}
