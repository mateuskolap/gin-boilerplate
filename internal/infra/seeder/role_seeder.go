package seeder

import (
	"context"
	"fmt"
	"gin-boilerplate/internal/domain"

	"gorm.io/gorm"
)

type roleSeeder struct{}

func NewRoleSeeder() Seeder {
	return &roleSeeder{}
}

func (s *roleSeeder) Seed(ctx context.Context, tx *gorm.DB) error {
	adminRole := domain.Role{Name: domain.RoleAdmin}
	if err := tx.WithContext(ctx).
		Where("name = ?", domain.RoleAdmin).
		FirstOrCreate(&adminRole).Error; err != nil {
		return fmt.Errorf("failed to get or create admin role: %w", err)
	}

	userRole := domain.Role{Name: domain.RoleUser}
	if err := tx.WithContext(ctx).
		Where("name = ?", domain.RoleUser).
		FirstOrCreate(&userRole).Error; err != nil {
		return fmt.Errorf("failed to get or create user role: %w", err)
	}

	var allPermissions []domain.Permission
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
