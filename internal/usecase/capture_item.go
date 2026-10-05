package usecase

import (
	"context"

	"github.com/gustavogmartinelli/gtd/internal/domain"
)

type CaptureItemInput struct {
	Title string
	Notes string
}

// CaptureItemUseCase adds a new item to the inbox: the GTD "capture" step,
// get it out of your head with zero friction.
type CaptureItemUseCase struct {
	repo ItemRepository
}

func NewCaptureItemUseCase(repo ItemRepository) *CaptureItemUseCase {
	return &CaptureItemUseCase{repo: repo}
}

func (uc *CaptureItemUseCase) Execute(ctx context.Context, in CaptureItemInput) (*domain.Item, error) {
	item, err := domain.NewItem(in.Title, in.Notes)
	if err != nil {
		return nil, err
	}
	if err := uc.repo.Save(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}
