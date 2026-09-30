package seeder

import (
	"context"
	"fmt"
	"gin-boilerplate/config"
	permissionpostgres "gin-boilerplate/internal/permissions/adapters/postgres"
	rolepostgres "gin-boilerplate/internal/roles/adapters/postgres"
	userpostgres "gin-boilerplate/internal/users/adapters/postgres"

	"gorm.io/gorm"
)

type Seeder interface {
	Seed(ctx context.Context, tx *gorm.DB) error
}

type DatabaseSeeder struct {
	db      *gorm.DB
	cfg     *config.Config
	seeders []Seeder
}

func NewDatabaseSeeder(
	db *gorm.DB,
	cfg *config.Config,
) *DatabaseSeeder {
	return &DatabaseSeeder{
		db:  db,
		cfg: cfg,
		seeders: []Seeder{
			permissionpostgres.NewPermissionSeeder(),
			rolepostgres.NewRoleSeeder(),
			userpostgres.NewUserSeeder(cfg),
		},
	}
}

func (s *DatabaseSeeder) Run(ctx context.Context) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, seeder := range s.seeders {
			if err := seeder.Seed(ctx, tx); err != nil {
				return fmt.Errorf("seeder error: %w", err)
			}
		}
		return nil
	})
}
