package application

import (
	"context"
	"gin-boilerplate/internal/domain/shared"

	"uuid"
)

type baseFindUseCase[T any] struct {
	repo interface {
		GetByID(context.Context, uuid.UUID) (*T, error)
	}
}

func NewBaseFindUseCase[T any](
	repo interface {
		GetByID(context.Context, uuid.UUID) (*T, error)
	},
) shared.BaseFindUseCase[T] {
	return &baseFindUseCase[T]{repo: repo}
}

func (uc *baseFindUseCase[T]) Find(ctx context.Context, id uuid.UUID) (*T, error) {
	return FindByID(ctx, uc.repo, id)
}

func FindByID[T any](ctx context.Context, repo interface {
	GetByID(context.Context, uuid.UUID) (*T, error)
}, id uuid.UUID) (*T, error) {
	return FindByIDUsing(ctx, id, repo.GetByID)
}

func FindByIDUsing[T any](ctx context.Context, id uuid.UUID, lookup func(context.Context, uuid.UUID) (*T, error)) (*T, error) {
	entity, err := lookup(ctx, id)
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
