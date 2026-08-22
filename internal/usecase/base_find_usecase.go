package usecase

import (
	"context"
	"gin-boilerplate/internal/domain"

	"github.com/google/uuid"
)

type baseFindUseCase[T any] struct {
	repo     domain.BaseRepository[T]
	preloads []string
}

func NewBaseFindUseCase[T any](
	repo domain.BaseRepository[T],
	preloads ...string,
) domain.BaseFindUseCase[T] {
	return &baseFindUseCase[T]{
		repo:     repo,
		preloads: preloads,
	}
}

func (uc *baseFindUseCase[T]) Find(ctx context.Context, id uuid.UUID) (*T, error) {
	entity, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to retrieve item",
			err,
		)
	}

	if entity == nil {
		return nil, domain.NewAppError(
			domain.ErrTypeNotFound,
			"Item not found",
			nil,
		)
	}

	return entity, nil
}
