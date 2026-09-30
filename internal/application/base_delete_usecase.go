package application

import (
	"context"
	"gin-boilerplate/internal/domain/shared"
	"uuid"
)

type baseDeleteUseCase[T any] struct {
	repo interface {
		Delete(context.Context, uuid.UUID) error
	}
	findUseCase shared.BaseFindUseCase[T]
}

func NewBaseDeleteUseCase[T any](
	repo interface {
		Delete(context.Context, uuid.UUID) error
		GetByID(context.Context, uuid.UUID) (*T, error)
	},
) shared.BaseDeleteUseCase {
	return &baseDeleteUseCase[T]{
		repo: repo,
		findUseCase: NewBaseFindUseCase[T](
			repo,
		),
	}
}

func (b *baseDeleteUseCase[T]) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := b.findUseCase.Find(ctx, id); err != nil {
		return err
	}

	if err := b.repo.Delete(ctx, id); err != nil {
		return shared.NewAppError(
			shared.ErrTypeInternal,
			"Failed to delete entity",
			err,
		)
	}

	return nil
}
