package usecase

import (
	"context"
	"gin-boilerplate/internal/domain/shared"
	"uuid"
)

type baseDeleteUseCase[T any] struct {
	repo        shared.BaseRepository[T]
	findUseCase shared.BaseFindUseCase[T]
}

func NewBaseDeleteUseCase[T any](
	repo shared.BaseRepository[T],
) shared.BaseDeleteUseCase {
	return &baseDeleteUseCase[T]{
		repo: repo,
		findUseCase: NewBaseFindUseCase(
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
