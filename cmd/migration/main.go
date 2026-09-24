package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"gin-boilerplate/config"

	migrate "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

const migrationsDirectory = "db/migrations"

func main() {
	if err := run(os.Args[1:], time.Now().UTC()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, now time.Time) error {
	if len(args) == 0 {
		return usageError()
	}

	switch args[0] {
	case "create":
		if len(args) != 2 {
			return usageError()
		}
		name, err := normalizeName(args[1])
		if err != nil {
			return err
		}
		return createMigration(migrationsDirectory, now.UTC(), name)
	case "up":
		if len(args) != 1 {
			return usageError()
		}
		return runMigrations("up", 0)
	case "down":
		steps := 1
		if len(args) == 2 {
			parsed, err := strconv.Atoi(args[1])
			if err != nil || parsed < 1 {
				return errors.New("down step count must be a positive integer")
			}
			steps = parsed
		} else if len(args) != 1 {
			return usageError()
		}
		return runMigrations("down", steps)
	case "version":
		if len(args) != 1 {
			return usageError()
		}
		return runMigrations("version", 0)
	default:
		return usageError()
	}
}

func usageError() error {
	return errors.New("usage: go run ./cmd/migration <create <name>|up|down [steps]|version>")
}

func runMigrations(action string, steps int) (resultErr error) {
	databaseConfig, err := config.LoadDatabaseConfig()
	if err != nil {
		return fmt.Errorf("load database configuration: %w", err)
	}

	db, err := sql.Open("postgres", databaseConfig.URL())
	if err != nil {
		return fmt.Errorf("open database connection: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("close database connection: %w", err))
		}
	}()

	pingContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = db.PingContext(pingContext)
	cancel()
	if err != nil {
		return fmt.Errorf("connect to PostgreSQL: %w", err)
	}

	databaseDriver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("initialize migration database driver: %w", err)
	}

	sourceDriver, err := iofs.New(os.DirFS("."), migrationsDirectory)
	if err != nil {
		_ = databaseDriver.Close()
		return fmt.Errorf("initialize migration files: %w", err)
	}

	migrator, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", databaseDriver)
	if err != nil {
		_ = sourceDriver.Close()
		_ = databaseDriver.Close()
		return fmt.Errorf("initialize migrator: %w", err)
	}
	defer func() {
		sourceErr, databaseErr := migrator.Close()
		if sourceErr != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("close migration source: %w", sourceErr))
		}
		if databaseErr != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("close migration driver: %w", databaseErr))
		}
	}()

	switch action {
	case "up":
		err = migrator.Up()
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Println("Database is already up to date")
			return nil
		}
		if err != nil {
			return fmt.Errorf("apply migrations: %w", err)
		}
		fmt.Println("Migrations applied")
	case "down":
		err = migrator.Steps(-steps)
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Println("No migrations to roll back")
			return nil
		}
		if err != nil {
			return fmt.Errorf("roll back migrations: %w", err)
		}
		fmt.Printf("Rolled back %d migration(s)\n", steps)
	case "version":
		version, dirty, err := migrator.Version()
		if errors.Is(err, migrate.ErrNilVersion) {
			fmt.Println("No migrations applied")
			return nil
		}
		if err != nil {
			return fmt.Errorf("read migration version: %w", err)
		}
		if dirty {
			return fmt.Errorf("database migration version %d is dirty", version)
		}
		fmt.Println(version)
	default:
		return usageError()
	}

	return nil
}

func normalizeName(value string) (string, error) {
	var normalized strings.Builder
	pendingSeparator := false

	for _, character := range strings.ToLower(strings.TrimSpace(value)) {
		switch {
		case character >= 'a' && character <= 'z', character >= '0' && character <= '9':
			if pendingSeparator && normalized.Len() > 0 {
				normalized.WriteByte('_')
			}
			normalized.WriteRune(character)
			pendingSeparator = false
		case character == '-' || character == '_' || unicode.IsSpace(character):
			pendingSeparator = normalized.Len() > 0
		default:
			return "", fmt.Errorf("migration name contains unsupported character %q", character)
		}
	}

	if normalized.Len() == 0 {
		return "", errors.New("migration name must contain at least one letter or number")
	}
	return normalized.String(), nil
}

func createMigration(directory string, now time.Time, name string) (resultErr error) {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create migrations directory: %w", err)
	}

	version := now.Format("20060102150405")
	matchingMigrations, err := filepath.Glob(filepath.Join(directory, version+"_*.sql"))
	if err != nil {
		return fmt.Errorf("check migration version: %w", err)
	}
	if len(matchingMigrations) > 0 {
		return fmt.Errorf("migration version %s already exists", version)
	}

	baseName := version + "_" + name
	upPath := filepath.Join(directory, baseName+".up.sql")
	downPath := filepath.Join(directory, baseName+".down.sql")
	var upCreated, downCreated bool
	defer func() {
		if resultErr == nil {
			return
		}
		if downCreated {
			resultErr = errors.Join(resultErr, removeFile(downPath))
		}
		if upCreated {
			resultErr = errors.Join(resultErr, removeFile(upPath))
		}
	}()

	upCreated, resultErr = createMigrationFile(upPath)
	if resultErr != nil {
		return resultErr
	}

	downCreated, resultErr = createMigrationFile(downPath)
	if resultErr != nil {
		return resultErr
	}

	return nil
}

func createMigrationFile(path string) (created bool, resultErr error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return false, fmt.Errorf("create migration file %q: %w", path, err)
	}
	created = true
	defer func() {
		if err := file.Close(); err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("close migration file %q: %w", path, err))
		}
	}()

	if _, err := io.WriteString(file, "-- Write migration SQL here.\n"); err != nil {
		return true, fmt.Errorf("write migration file %q: %w", path, err)
	}
	return true, nil
}

func removeFile(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove incomplete migration file %q: %w", path, err)
	}
	return nil
}
