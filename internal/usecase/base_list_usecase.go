package usecase

import (
	"context"
	"gin-boilerplate/internal/domain"
)

type baseListUseCase[T any] struct {
	repo          domain.BaseRepository[T]
	allowedFields map[string]bool
	preloads      []string
}

func NewBaseListUseCase[T any](
	repo domain.BaseRepository[T],
	allowedFields map[string]bool,
	preloads ...string,
) domain.BaseListUseCase[T] {
	return &baseListUseCase[T]{
		repo:          repo,
		allowedFields: allowedFields,
		preloads:      preloads,
	}
}

func (uc *baseListUseCase[T]) List(
	ctx context.Context,
	params domain.PaginationParams,
	filters []domain.Filter,
) (*domain.PaginatedResult[T], error) {
	if err := domain.Filters(filters).ValidateAllowed(uc.allowedFields); err != nil {
		return nil, err
	}

	if err := params.ValidateSort(uc.allowedFields); err != nil {
		return nil, err
	}

	result, err := uc.repo.List(ctx, params, filters, uc.preloads...)
	if err != nil {
		return nil, domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to retrieve items",
			err,
		)
	}
	return result, nil
}
