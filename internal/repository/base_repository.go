package repository

import (
	"context"
	"errors"
	"fmt"
	"gin-boilerplate/internal/domain"
	"math"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type baseRepository[T any] struct {
	db *gorm.DB
}

func NewBaseRepository[T any](db *gorm.DB) domain.BaseRepository[T] {
	return &baseRepository[T]{
		db: db,
	}
}

func (r *baseRepository[T]) Create(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

func (r *baseRepository[T]) GetByID(ctx context.Context, id uuid.UUID, preloads ...string) (*T, error) {
	var entity T

	query := r.db.WithContext(ctx)

	for _, preload := range preloads {
		query = query.Preload(preload)
	}

	err := query.Where("id = ?", id).First(&entity).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &entity, nil
}

func (r *baseRepository[T]) Update(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Save(entity).Error
}

func (r *baseRepository[T]) Delete(ctx context.Context, id uuid.UUID) error {
	var entity T
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&entity).Error
}

func (r *baseRepository[T]) List(
	ctx context.Context,
	params domain.PaginationParams,
	filters []domain.Filter,
) (*domain.PaginatedResult[T], error) {
	var entities []*T
	var total int64

	query := r.db.WithContext(ctx).Model(new(T))

	for _, preload := range params.Preloads {
		query = query.Preload(preload)
	}

	joinedRelations := make(map[string]bool)

	for _, f := range filters {
		var clause string
		var fieldName string = f.Field

		if strings.Contains(f.Field, ".") {
			parts := strings.Split(f.Field, ".")

			column := parts[len(parts)-1]

			var currentPath string
			var lastRelation string

			for i := 0; i < len(parts)-1; i++ {
				if currentPath == "" {
					currentPath = parts[i]
				} else {
					currentPath = currentPath + "." + parts[i]
				}

				if !joinedRelations[currentPath] {
					query = query.Joins(currentPath)
					joinedRelations[currentPath] = true
				}

				lastRelation = parts[i]
			}

			fieldName = fmt.Sprintf(`"%s.%s"`, lastRelation, column)
		}

		if f.Operator == domain.OperatorIn || f.Operator == domain.OperatorNotIn {
			clause = fmt.Sprintf("%s %s (?)", fieldName, string(f.Operator))
		} else {
			clause = fmt.Sprintf("%s %s ?", fieldName, string(f.Operator))
		}

		query = query.Where(clause, f.Value)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	if params.Page <= 0 {
		params.Page = 1
	}

	if params.Limit <= 0 || params.Limit > 100 {
		params.Limit = 10
	}

	offset := (params.Page - 1) * params.Limit

	if err := query.Offset(offset).Limit(params.Limit).Find(&entities).Error; err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))

	return &domain.PaginatedResult[T]{
		Items:      entities,
		Total:      total,
		Page:       params.Page,
		Limit:      params.Limit,
		TotalPages: totalPages,
	}, nil
}
