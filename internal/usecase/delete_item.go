package usecase

import "context"

// DeleteItemUseCase permanently removes an item.
type DeleteItemUseCase struct {
	repo ItemRepository
}

func NewDeleteItemUseCase(repo ItemRepository) *DeleteItemUseCase {
	return &DeleteItemUseCase{repo: repo}
}

func (uc *DeleteItemUseCase) Execute(ctx context.Context, id int64) error {
	return uc.repo.Delete(ctx, id)
}
