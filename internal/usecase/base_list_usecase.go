package usecase

import (
	"context"
	"fmt"
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
	for _, f := range filters {
		if !uc.allowedFields[f.Field] {
			return nil, domain.NewAppError(
				domain.ErrTypeValidation,
				fmt.Sprintf("filtering by field '%s' is not allowed", f.Field),
				nil,
			)
		}
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
