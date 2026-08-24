package repository

import (
	"context"
	"errors"
	"fmt"
	"gin-boilerplate/internal/domain"
	"math"
	"strings"

	"uuid"
	"gorm.io/gorm"
)

type baseRepository[T any] struct {
	db *gorm.DB
}

func NewBaseRepository[T any](db *gorm.DB) domain.BaseRepository[T] {
	return &baseRepository[T]{db: db}
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

	if err := query.Where("id = ?", id).First(&entity).Error; err != nil {
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
	preloads ...string,
) (*domain.PaginatedResult[T], error) {
	params.Sanitize()

	query := r.db.WithContext(ctx).Model(new(T))

	for _, p := range preloads {
		query = query.Preload(p)
	}

	query, err := applyFilters(query, filters)
	if err != nil {
		return nil, err
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	for _, s := range params.Sort {
		query = query.Order(fmt.Sprintf("%s %s", s.Field, s.Direction))
	}

	var entities []*T
	if err := query.Offset(params.Offset()).Limit(params.Limit).Find(&entities).Error; err != nil {
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

func applyFilters(query *gorm.DB, filters []domain.Filter) (*gorm.DB, error) {
	joined := make(map[string]bool)

	for _, f := range filters {
		if err := f.Validate(); err != nil {
			return nil, err
		}

		field := f.Field

		if strings.Contains(f.Field, ".") {
			parts := strings.Split(f.Field, ".")
			column := parts[len(parts)-1]

			var currentPath string
			var lastRelation string

			for i := 0; i < len(parts)-1; i++ {
				if currentPath == "" {
					currentPath = parts[i]
				} else {
					currentPath += "." + parts[i]
				}

				if !joined[currentPath] {
					query = query.Joins(currentPath)
					joined[currentPath] = true
				}

				lastRelation = parts[i]
			}

			field = fmt.Sprintf(`"%s"."%s"`, lastRelation, column)
		}

		if f.IsSetOperator() {
			query = query.Where(fmt.Sprintf("%s %s (?)", field, string(f.Operator)), f.Value)
		} else {
			query = query.Where(fmt.Sprintf("%s %s ?", field, string(f.Operator)), f.Value)
		}
	}

	return query, nil
}
