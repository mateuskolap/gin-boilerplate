package application

import (
	"context"
	"gin-boilerplate/internal/domain/shared"
)

type baseListUseCase[T any] struct {
	repo interface {
		List(context.Context, shared.PaginationParams, []shared.Filter) (*shared.PaginatedResult[T], error)
	}
	allowedFields map[string]bool
}

func NewBaseListUseCase[T any](
	repo interface {
		List(context.Context, shared.PaginationParams, []shared.Filter) (*shared.PaginatedResult[T], error)
	},
	allowedFields map[string]bool,
) shared.BaseListUseCase[T] {
	return &baseListUseCase[T]{
		repo:          repo,
		allowedFields: allowedFields,
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

	result, err := uc.repo.List(ctx, params, filters)
	if err != nil {
		return nil, shared.NewAppError(
			shared.ErrTypeInternal,
			"Failed to retrieve items",
			err,
		)
	}
	return result, nil
}
