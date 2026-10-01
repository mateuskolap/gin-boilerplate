package bootstrap

import (
	"context"
	"errors"
	"time"

	"gin-boilerplate/config"
	"gin-boilerplate/internal/infra/seeder"
)

// RunSeed opens only PostgreSQL; it does not construct HTTP, storage or Redis dependencies.
func RunSeed(ctx context.Context, cfg *config.Config) (resultErr error) {
	if err := cfg.ValidateSeeder(); err != nil {
		return err
	}
	db, sqlDB, err := openDatabase(cfg)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, sqlDB.Close()) }()
	seedContext, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return seeder.NewDatabaseSeeder(db, cfg).Run(seedContext)
}
