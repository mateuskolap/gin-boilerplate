package main

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestNormalizeMigrationName(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  string
	}{
		{input: " Add User-Role ", want: "add_user_role"},
		{input: "refresh__tokens", want: "refresh_tokens"},
		{input: "V2 Schema", want: "v2_schema"},
	} {
		got, err := normalizeName(tc.input)
		if err != nil || got != tc.want {
			t.Errorf("normalizeName(%q) = %q, %v; want %q", tc.input, got, err, tc.want)
		}
	}
	for _, input := range []string{"", "---", "user/table", "café"} {
		if _, err := normalizeName(input); err == nil {
			t.Errorf("normalizeName(%q) accepted invalid input", input)
		}
	}
}

func TestMigrationCLIRejectsInvalidCommandsBeforeDatabaseAccess(t *testing.T) {
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	for _, args := range [][]string{
		nil,
		{"unknown"},
		{"up", "extra"},
		{"down", "0"},
		{"down", "not-a-number"},
		{"create"},
		{"create", "bad/name"},
	} {
		if err := run(args, now); err == nil {
			t.Errorf("run(%q) returned nil", args)
		}
	}
}

func TestCreateMigrationWritesPairedTemplatesAndPreventsDuplicateVersion(t *testing.T) {
	directory := t.TempDir()
	now := time.Date(2026, 9, 27, 12, 34, 56, 0, time.UTC)
	if err := createMigration(directory, now, "add_users"); err != nil {
		t.Fatalf("createMigration() error = %v", err)
	}
	for _, suffix := range []string{"up", "down"} {
		path := filepath.Join(directory, "20260927123456_add_users."+suffix+".sql")
		data, err := os.ReadFile(path)
		if err != nil || string(data) != "-- Write migration SQL here.\n" {
			t.Errorf("migration file %q content=%q error=%v", path, data, err)
		}
	}
	if err := createMigration(directory, now, "another_change"); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("createMigration() duplicate version error = %v", err)
	}
}

func TestCreateMigrationDoesNotOverwriteExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "existing.sql")
	if err := os.WriteFile(path, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := createMigrationFile(path); !errors.Is(err, os.ErrExist) {
		t.Fatalf("createMigrationFile() error = %v, want file exists", err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "keep" {
		t.Fatalf("existing file content=%q error=%v", data, err)
	}
}

func TestCreateMigrationRejectsExistingVersionWithoutOverwrite(t *testing.T) {
	directory := t.TempDir()
	now := time.Date(2026, 9, 27, 12, 34, 56, 0, time.UTC)
	downPath := filepath.Join(directory, "20260927123456_partial.down.sql")
	if err := os.WriteFile(downPath, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := createMigration(directory, now, "partial"); err == nil {
		t.Fatal("createMigration() overwrote the existing down migration")
	}
	upPath := filepath.Join(directory, "20260927123456_partial.up.sql")
	if _, err := os.Stat(upPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial up migration still exists: %v", err)
	}
	if content, err := os.ReadFile(downPath); err != nil || string(content) != "keep" {
		t.Fatalf("existing down migration changed: %q, %v", content, err)
	}
}

func TestRemoveFileDeletesExistingFileAndIgnoresMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "migration.sql")
	if err := os.WriteFile(path, []byte("migration"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := removeFile(path); err != nil {
		t.Fatalf("removeFile() error = %v", err)
	}
	if err := removeFile(path); err != nil {
		t.Fatalf("removeFile() missing file error = %v", err)
	}
}

func TestRunMigrationsAgainstDisposableDatabase(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to a disposable PostgreSQL database")
	}
	parsed, err := url.Parse(dsn)
	if err != nil || parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		t.Fatal("TEST_DATABASE_URL must be a PostgreSQL URL")
	}
	databaseName := strings.Trim(parsed.Path, "/")
	lowerDatabaseName := strings.ToLower(databaseName)
	if lowerDatabaseName != "test" && !strings.HasSuffix(lowerDatabaseName, "_test") && !strings.HasSuffix(lowerDatabaseName, "-test") {
		t.Fatal("TEST_DATABASE_URL database must be named 'test' or end in '_test' or '-test'")
	}
	username := ""
	password := ""
	if parsed.User != nil {
		username = parsed.User.Username()
		password, _ = parsed.User.Password()
	}
	port, err := strconv.Atoi(parsed.Port())
	if err != nil {
		t.Fatalf("TEST_DATABASE_URL must include a port: %v", err)
	}
	for key, value := range map[string]string{
		"ENVIRONMENT": "test", "DB_HOST": parsed.Hostname(), "DB_PORT": strconv.Itoa(port),
		"DB_USER": username, "DB_PASSWORD": password, "DB_NAME": databaseName, "DB_SSLMODE": "disable",
	} {
		t.Setenv(key, value)
	}

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate migration test source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "../.."))
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("change to project root: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })

	if err := runMigrations("up", 0); err != nil {
		t.Fatalf("run migration up: %v", err)
	}
	if err := runMigrations("version", 0); err != nil {
		t.Fatalf("read migration version: %v", err)
	}
}
