package seeder

import (
	"context"
	"fmt"
	"gin-boilerplate/config"
	"gin-boilerplate/internal/domain"

	"gorm.io/gorm"
)

type Seeder interface {
	Seed(ctx context.Context, tx *gorm.DB) error
}

type DatabaseSeeder struct {
	db                 *gorm.DB
	cfg                *config.Config
	rolePermissionRepo domain.RolePermissionRepository
	seeders            []Seeder
}

func NewDatabaseSeeder(
	db *gorm.DB,
	cfg *config.Config,
	rolePermissionRepo domain.RolePermissionRepository,
) *DatabaseSeeder {
	return &DatabaseSeeder{
		db:                 db,
		cfg:                cfg,
		rolePermissionRepo: rolePermissionRepo,
		seeders: []Seeder{
			NewPermissionSeeder(),
			NewRoleSeeder(),
			NewUserSeeder(cfg),
		},
	}
}

func (s *DatabaseSeeder) Run(ctx context.Context) error {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, seeder := range s.seeders {
			if err := seeder.Seed(ctx, tx); err != nil {
				return fmt.Errorf("seeder error: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	if s.rolePermissionRepo != nil {
		return s.rolePermissionRepo.InvalidateAll(ctx)
	}

	return nil
}
