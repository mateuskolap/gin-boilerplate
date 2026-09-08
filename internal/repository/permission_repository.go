package repository

import (
	"context"
	"gin-boilerplate/internal/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type permissionRepository struct {
	domain.BaseRepository[domain.Permission]
	db *gorm.DB
}

func NewPermissionRepository(db *gorm.DB) domain.PermissionRepository {
	return &permissionRepository{
		BaseRepository: NewBaseRepository[domain.Permission](db),
		db:             db,
	}
}

func (p *permissionRepository) UpsertByName(ctx context.Context, permissions []domain.Permission) error {
	return p.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoNothing: true,
	}).Create(&permissions).Error
}

func (p *permissionRepository) DeleteByNames(ctx context.Context, names []string) error {
	return p.db.WithContext(ctx).Where("name IN ?", names).Delete(&domain.Permission{}).Error
}

func (p *permissionRepository) ListByRoleName(ctx context.Context, roleName string, params domain.PaginationParams, filters []domain.Filter) (*domain.PaginatedResult[domain.Permission], error) {
	sanitizedFilters := append(domain.Filters(filters).Without("role_name"), domain.Filter{
		Field:    "name",
		Operator: domain.OperatorEquals,
		Value:    roleName,
	})

	return p.List(ctx, params, sanitizedFilters)
}
