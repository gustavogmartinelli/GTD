package usecase

import (
	"context"

	"github.com/gustavogmartinelli/gtd/internal/domain"
)

type ProcessItemInput struct {
	ID      int64
	Context string
	Notes   string
}

// ProcessItemUseCase implements the GTD "clarify" step: turn an inbox
// item into a next action, tagged with the context needed to do it.
type ProcessItemUseCase struct {
	repo ItemRepository
}

func NewProcessItemUseCase(repo ItemRepository) *ProcessItemUseCase {
	return &ProcessItemUseCase{repo: repo}
}

func (uc *ProcessItemUseCase) Execute(ctx context.Context, in ProcessItemInput) (*domain.Item, error) {
	item, err := uc.repo.FindByID(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	item.Process(in.Context, in.Notes)
	if err := uc.repo.Update(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}
