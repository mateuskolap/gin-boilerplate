package usecase

import (
	"context"
	"gin-boilerplate/internal/domain/shared"
)

type baseListUseCase[T any] struct {
	repo          shared.BaseRepository[T]
	allowedFields map[string]bool
	preloads      []string
}

func NewBaseListUseCase[T any](
	repo shared.BaseRepository[T],
	allowedFields map[string]bool,
	preloads ...string,
) shared.BaseListUseCase[T] {
	return &baseListUseCase[T]{
		repo:          repo,
		allowedFields: allowedFields,
		preloads:      preloads,
	}
}

func (uc *baseListUseCase[T]) List(
	ctx context.Context,
	params shared.PaginationParams,
	filters []shared.Filter,
) (*shared.PaginatedResult[T], error) {
	if err := shared.Filters(filters).ValidateAllowed(uc.allowedFields); err != nil {
		return nil, err
	}

	if err := params.ValidateSort(uc.allowedFields); err != nil {
		return nil, err
	}

	result, err := uc.repo.List(ctx, params, filters, uc.preloads...)
	if err != nil {
		return nil, shared.NewAppError(
			shared.ErrTypeInternal,
			"Failed to retrieve items",
			err,
		)
	}
	return result, nil
}
