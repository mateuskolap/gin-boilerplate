package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"gin-boilerplate/config"
	"gin-boilerplate/internal/infra/migrations"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 1 || (args[0] != "up" && args[0] != "down") {
		return errors.New("usage: go run ./cmd/migrate <up|down>")
	}

	databaseConfig, err := config.LoadDatabaseConfig()
	if err != nil {
		return fmt.Errorf("load database configuration: %w", err)
	}

	db, err := openDatabase(databaseConfig)
	if err != nil {
		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get database connection: %w", err)
	}
	defer sqlDB.Close()

	pingContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingContext); err != nil {
		return fmt.Errorf("connect to PostgreSQL: %w", err)
	}

	switch args[0] {
	case "up":
		if err := withMigrationLock(db, migrations.Migrate); err != nil {
			return fmt.Errorf("apply migrations: %w", err)
		}
		fmt.Println("Migrations applied")
	case "down":
		if err := withMigrationLock(db, migrations.RollbackLast); err != nil {
			return fmt.Errorf("roll back migration: %w", err)
		}
		fmt.Println("Last migration rolled back")
	}

	return nil
}

func openDatabase(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.URL()), &gorm.Config{
		TranslateError:       true,
		DisableAutomaticPing: true,
	})
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL connection: %w", err)
	}
	return db, nil
}

func withMigrationLock(db *gorm.DB, run func(*gorm.DB) error) error {
	const advisoryLockID int64 = 0x47494E4D49475241

	return db.Connection(func(conn *gorm.DB) (migrationErr error) {
		if err := conn.Exec("SELECT pg_advisory_lock(?)", advisoryLockID).Error; err != nil {
			return fmt.Errorf("acquire migration lock: %w", err)
		}

		defer func() {
			if err := conn.Exec("SELECT pg_advisory_unlock(?)", advisoryLockID).Error; err != nil {
				migrationErr = errors.Join(migrationErr, fmt.Errorf("release migration lock: %w", err))
			}
		}()

		return run(conn)
	})
}
