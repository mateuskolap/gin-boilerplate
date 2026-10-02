package postgres

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

type BaseRepository[Entity any, Model any] struct {
	db        *gorm.DB
	newModel  func() Model
	toModel   func(*Entity) Model
	fromModel func(Model) *Entity
}

func NewBaseRepository[Entity any, Model any](
	db *gorm.DB,
	newModel func() Model,
	toModel func(*Entity) Model,
	fromModel func(Model) *Entity,
) *BaseRepository[Entity, Model] {
	return &BaseRepository[Entity, Model]{db: db, newModel: newModel, toModel: toModel, fromModel: fromModel}
}

func (r *BaseRepository[Entity, Model]) DB(ctx context.Context) *gorm.DB {
	return GetTxFromContext(ctx, r.db)
}

func (r *BaseRepository[Entity, Model]) Create(ctx context.Context, entity *Entity) error {
	model := r.toModel(entity)
	if err := translateError(r.DB(ctx).WithContext(ctx).Create(model).Error); err != nil {
		return err
	}
	*entity = *r.fromModel(model)
	return nil
}

func (r *BaseRepository[Entity, Model]) GetByID(ctx context.Context, id uuid.UUID) (*Entity, error) {
	return r.FindOneBy(ctx, "id = ?", []any{id})
}

func (r *BaseRepository[Entity, Model]) FindOneBy(ctx context.Context, query string, args []any, preloads ...string) (*Entity, error) {
	model := r.newModel()
	dbQuery := r.DB(ctx).WithContext(ctx).Model(model).Where(query, args...)

	for _, preload := range preloads {
		dbQuery = dbQuery.Preload(preload, nil)
	}

	if err := dbQuery.First(model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, translateError(err)
	}

	return r.fromModel(model), nil
}

func (r *BaseRepository[Entity, Model]) Update(ctx context.Context, entity *Entity) error {
	model := r.toModel(entity)
	if err := translateError(r.DB(ctx).WithContext(ctx).
		Model(model).
		Select("*").
		Omit("ID", "CreatedAt").
		Updates(model).Error); err != nil {
		return err
	}
	*entity = *r.fromModel(model)
	return nil
}

func translateError(err error) error {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return fmt.Errorf("%w: %w", shared.ErrConflict, err)
	}
	return err
}

func (r *BaseRepository[Entity, Model]) Delete(ctx context.Context, id uuid.UUID) error {
	model := r.newModel()
	return translateError(r.DB(ctx).WithContext(ctx).Where("id = ?", id).Delete(model).Error)
}

func (r *BaseRepository[Entity, Model]) List(
	ctx context.Context,
	params shared.PaginationParams,
	filters []shared.Filter,
) (*shared.PaginatedResult[Entity], error) {
	params.Sanitize()

	query := r.DB(ctx).WithContext(ctx).Model(r.newModel())

	query, err := applyFilters(query, filters)
	if err != nil {
		return nil, err
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, translateError(err)
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

	models := make([]Model, 0)
	if err := query.Offset(params.Offset()).Limit(params.Limit).Find(&models).Error; err != nil {
		return nil, translateError(err)
	}
	entities := make([]*Entity, len(models))
	for i, model := range models {
		entities[i] = r.fromModel(model)
	}

	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))

	return &shared.PaginatedResult[Entity]{
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
