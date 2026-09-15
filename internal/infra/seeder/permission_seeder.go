package seeder

import (
	"context"
	"fmt"
	"gin-boilerplate/internal/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type permissionSeeder struct{}

func NewPermissionSeeder() Seeder {
	return &permissionSeeder{}
}

func (s *permissionSeeder) Seed(ctx context.Context, tx *gorm.DB) error {
	validNames := make([]string, 0, len(domain.AllPermissions))
	permissions := make([]domain.Permission, 0, len(domain.AllPermissions))

	for _, p := range domain.AllPermissions {
		name := string(p)
		validNames = append(validNames, name)
		permissions = append(permissions, domain.Permission{
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
			Delete(&domain.Permission{}).Error; err != nil {
			return fmt.Errorf("failed to delete obsolete permissions: %w", err)
		}
	}

	return nil
}
