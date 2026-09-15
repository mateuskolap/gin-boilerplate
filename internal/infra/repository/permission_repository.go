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

