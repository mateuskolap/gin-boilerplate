package repository

import (
	"context"
	"gin-boilerplate/internal/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type permissionRepository struct {
	*baseRepository[domain.Permission]
}

func NewPermissionRepository(db *gorm.DB) domain.PermissionRepository {
	return &permissionRepository{
		baseRepository: newBaseRepository[domain.Permission](db),
	}
}

func (p *permissionRepository) UpsertByName(ctx context.Context, permissions []domain.Permission) error {
	return p.getDB(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoNothing: true,
	}).Create(&permissions).Error
}
