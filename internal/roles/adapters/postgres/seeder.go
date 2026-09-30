package postgres

import (
	"context"
	"fmt"
	permissionpostgres "gin-boilerplate/internal/permissions/adapters/postgres"
	roledomain "gin-boilerplate/internal/roles/domain"

	"gorm.io/gorm"
)

type roleSeeder struct{}

func NewRoleSeeder() interface {
	Seed(context.Context, *gorm.DB) error
} {
	return &roleSeeder{}
}

func (s *roleSeeder) Seed(ctx context.Context, tx *gorm.DB) error {
	adminRole := RoleModel{Name: roledomain.RoleAdmin}
	if err := tx.WithContext(ctx).
		Where("name = ?", roledomain.RoleAdmin).
		FirstOrCreate(&adminRole).Error; err != nil {
		return fmt.Errorf("failed to get or create admin role: %w", err)
	}

	userRole := RoleModel{Name: roledomain.RoleUser}
	if err := tx.WithContext(ctx).
		Where("name = ?", roledomain.RoleUser).
		FirstOrCreate(&userRole).Error; err != nil {
		return fmt.Errorf("failed to get or create user role: %w", err)
	}

	var allPermissions []permissionpostgres.PermissionModel
	if err := tx.WithContext(ctx).Find(&allPermissions).Error; err != nil {
		return fmt.Errorf("failed to fetch all permissions: %w", err)
	}

	if err := tx.WithContext(ctx).
		Model(&adminRole).
		Association("Permissions").
		Replace(&allPermissions); err != nil {
		return fmt.Errorf("failed to assign permissions to admin role: %w", err)
	}

	return nil
}
