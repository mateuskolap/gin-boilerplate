package usecase

import (
	"context"
	"gin-boilerplate/internal/domain/shared"

	"uuid"
)

type baseFindUseCase[T any] struct {
	repo     shared.BaseRepository[T]
	preloads []string
}

func NewBaseFindUseCase[T any](
	repo shared.BaseRepository[T],
	preloads ...string,
) shared.BaseFindUseCase[T] {
	return &baseFindUseCase[T]{
		repo:     repo,
		preloads: preloads,
	}
}

func (uc *baseFindUseCase[T]) Find(ctx context.Context, id uuid.UUID) (*T, error) {
	entity, err := uc.repo.GetByID(ctx, id, uc.preloads...)
	if err != nil {
		return nil, shared.NewAppError(
			shared.ErrTypeInternal,
			"Failed to retrieve item",
			err,
		)
	}

	if entity == nil {
		return nil, shared.NewAppError(
			shared.ErrTypeNotFound,
			"Item not found",
			nil,
		)
	}

	return entity, nil
}
