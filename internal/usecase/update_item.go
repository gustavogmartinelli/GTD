package usecase

import (
	"context"

	"github.com/gustavogmartinelli/gtd/internal/domain"
)

// UpdateItemInput carries partial changes; nil fields are left unchanged.
type UpdateItemInput struct {
	ID      int64
	Title   *string
	Notes   *string
	Status  *domain.Status
	Context *string
}

// UpdateItemUseCase applies partial edits to an existing item, delegating
// invariant checks to the domain entity itself.
type UpdateItemUseCase struct {
	repo ItemRepository
}

func NewUpdateItemUseCase(repo ItemRepository) *UpdateItemUseCase {
	return &UpdateItemUseCase{repo: repo}
}

func (uc *UpdateItemUseCase) Execute(ctx context.Context, in UpdateItemInput) (*domain.Item, error) {
	item, err := uc.repo.FindByID(ctx, in.ID)
	if err != nil {
		return nil, err
	}

	if in.Title != nil {
		if err := item.Rename(*in.Title); err != nil {
			return nil, err
		}
	}
	if in.Notes != nil {
		item.SetNotes(*in.Notes)
	}
	if in.Status != nil {
		if err := item.SetStatus(*in.Status); err != nil {
			return nil, err
		}
	}
	if in.Context != nil {
		item.SetContext(*in.Context)
	}

	if err := uc.repo.Update(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}
