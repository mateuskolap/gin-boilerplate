package usecase

import (
	"context"
	"gin-boilerplate/internal/domain"
	"uuid"
)

type baseDeleteUseCase[T any] struct {
	repo        domain.BaseRepository[T]
	findUseCase domain.BaseFindUseCase[T]
}

func NewBaseDeleteUseCase[T any](
	repo domain.BaseRepository[T],
) domain.BaseDeleteUseCase {
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
		return domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to delete entity",
			err,
		)
	}

	return nil
}
