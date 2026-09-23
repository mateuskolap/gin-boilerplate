package seeder

import (
	"context"
	"fmt"
	"gin-boilerplate/config"

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
			NewPermissionSeeder(),
			NewRoleSeeder(),
			NewUserSeeder(cfg),
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
