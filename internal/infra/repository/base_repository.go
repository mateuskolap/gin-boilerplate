package repository

import (
	"context"
	"errors"
	"fmt"
	"gin-boilerplate/internal/domain/shared"
	"math"
	"regexp"
	"strings"

	"uuid"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var identifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type baseRepository[T any] struct {
	db *gorm.DB
}

func newBaseRepository[T any](db *gorm.DB) *baseRepository[T] {
	return &baseRepository[T]{db: db}
}

func (r *baseRepository[T]) getDB(ctx context.Context) *gorm.DB {
	return GetTxFromContext(ctx, r.db)
}

func (r *baseRepository[T]) Create(ctx context.Context, entity *T) error {
	return r.getDB(ctx).Create(entity).Error
}

func (r *baseRepository[T]) GetByID(ctx context.Context, id uuid.UUID, preloads ...string) (*T, error) {
	return r.FindOneBy(ctx, "id = ?", []any{id}, preloads...)
}

func (r *baseRepository[T]) FindOneBy(ctx context.Context, query string, args []any, preloads ...string) (*T, error) {
	var entity T

	dbQuery := r.getDB(ctx)

	for _, preload := range preloads {
		dbQuery = dbQuery.Preload(preload)
	}

	if err := dbQuery.Where(query, args...).First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &entity, nil
}

func (r *baseRepository[T]) Update(ctx context.Context, entity *T) error {
	return r.getDB(ctx).
		Model(entity).
		Select("*").
		Omit("ID", "CreatedAt").
		Updates(entity).Error
}

func (r *baseRepository[T]) Delete(ctx context.Context, id uuid.UUID) error {
	var entity T
	return r.getDB(ctx).Where("id = ?", id).Delete(&entity).Error
}

func (r *baseRepository[T]) List(
	ctx context.Context,
	params shared.PaginationParams,
	filters []shared.Filter,
	preloads ...string,
) (*shared.PaginatedResult[T], error) {
	params.Sanitize()

	query := r.getDB(ctx).Model(new(T))

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

	containsIDSort := false
	for _, sort := range params.Sort {
		query = query.Order(clause.OrderByColumn{
			Column: clause.Column{Name: sort.Field},
			Desc:   sort.Direction == shared.SortDesc,
		})
		containsIDSort = containsIDSort || sort.Field == "id"
	}
	if !containsIDSort {
		query = query.Order(clause.OrderByColumn{Column: clause.Column{Name: "id"}})
	}

	var entities []*T
	if err := query.Offset(params.Offset()).Limit(params.Limit).Find(&entities).Error; err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))

	return &shared.PaginatedResult[T]{
		Items:      entities,
		Total:      total,
		Page:       params.Page,
		Limit:      params.Limit,
		TotalPages: totalPages,
	}, nil
}

func applyFilters(query *gorm.DB, filters []shared.Filter) (*gorm.DB, error) {
	joined := make(map[string]bool)

	for _, f := range filters {
		if err := f.Validate(); err != nil {
			return nil, err
		}

		field := f.Field
		if !isSafeIdentifierPath(field) {
			return nil, fmt.Errorf("invalid filter field: %q", field)
		}

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
		} else {
			field = fmt.Sprintf(`"%s"`, field)
		}

		if f.IsSetOperator() {
			query = query.Where(fmt.Sprintf("%s %s (?)", field, string(f.Operator)), f.Value)
		} else if f.IsNullOperator() {
			query = query.Where(fmt.Sprintf("%s %s", field, string(f.Operator)))
		} else {
			query = query.Where(fmt.Sprintf("%s %s ?", field, string(f.Operator)), f.Value)
		}
	}

	return query, nil
}

func isSafeIdentifierPath(value string) bool {
	for _, part := range strings.Split(value, ".") {
		if !identifierPattern.MatchString(part) {
			return false
		}
	}
	return true
}
