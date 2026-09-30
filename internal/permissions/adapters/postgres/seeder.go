package postgres

import (
	"context"
	"fmt"
	permissiondomain "gin-boilerplate/internal/permissions/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type permissionSeeder struct{}

func NewPermissionSeeder() interface {
	Seed(context.Context, *gorm.DB) error
} {
	return &permissionSeeder{}
}

func (s *permissionSeeder) Seed(ctx context.Context, tx *gorm.DB) error {
	validNames := make([]string, 0, len(permissiondomain.AllPermissions))
	permissions := make([]PermissionModel, 0, len(permissiondomain.AllPermissions))

	for _, p := range permissiondomain.AllPermissions {
		name := string(p)
		validNames = append(validNames, name)
		permissions = append(permissions, PermissionModel{
			Name: name,
		})
	}

	if len(permissions) > 0 {
		if err := tx.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "name"}},
			DoNothing: true,
		}).Create(&permissions).Error; err != nil {
			return fmt.Errorf("failed to upsert permissions: %w", err)
		}
	}

	if len(validNames) > 0 {
		if err := tx.WithContext(ctx).
			Where("name NOT IN ?", validNames).
			Delete(&PermissionModel{}).Error; err != nil {
			return fmt.Errorf("failed to delete obsolete permissions: %w", err)
		}
	}

	return nil
}
