package cli

import (
	"context"
	"errors"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHelpAndArgumentValidationDoNotInitializeServices(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("JWT_SECRET", "")
	t.Setenv("DB_PORT", "invalid")
	for _, args := range [][]string{nil, {"help"}, {"--help"}, {"help", "dev"}, {"db:seed", "--help"}} {
		if err := Run(context.Background(), args); err != nil {
			t.Fatalf("help %v: %v", args, err)
		}
	}
	invalidArgs := [][]string{
		{"unknown"},
		{"serve", "--seed"},
		{"dev", "--unknown"},
		{"test", "extra"},
		{"make:domain", "bad/name"},
		{"make:domain", "Project", "--unknown"},
		{"make:domain", "Project", "Other"},
	}
	for _, args := range invalidArgs {
		if err := Run(context.Background(), args); err == nil || ExitCode(err) != 2 {
			t.Fatalf("%v should fail with argument exit code 2, got %v", args, err)
		}
	}
}

func TestMakeDomainCLIParsesSoftDeleteFlagAfterDomain(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if err := os.Mkdir("internal", 0o755); err != nil {
		t.Fatal(err)
	}

	if err := Run(context.Background(), []string{"make:domain", "Invoice", "--soft-delete"}); err != nil {
		t.Fatalf("make:domain with soft-delete: %v", err)
	}
	migrations, err := filepath.Glob(filepath.Join(root, migrationsDirectory, "*_create_invoices.up.sql"))
	if err != nil || len(migrations) != 1 {
		t.Fatalf("generated migration paths=%v error=%v", migrations, err)
	}
	upSQL, err := os.ReadFile(migrations[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(upSQL), "deleted_at TIMESTAMPTZ") {
		t.Fatalf("soft-delete migration does not include deleted_at: %s", upSQL)
	}
}

func TestMakeDomainCLIParsesActivityLogsFlagAfterDomain(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if err := os.Mkdir("internal", 0o755); err != nil {
		t.Fatal(err)
	}

	if err := Run(context.Background(), []string{"make:domain", "Invoice", "--soft-delete", "--activity-logs"}); err != nil {
		t.Fatalf("make:domain with activity-logs: %v", err)
	}

	feature := filepath.Join(root, "internal", "invoices")
	domain, err := os.ReadFile(filepath.Join(feature, "domain", "invoice.go"))
	if err != nil {
		t.Fatal(err)
	}
	repository, err := os.ReadFile(filepath.Join(feature, "adapters", "postgres", "invoice_repository.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range [][]byte{domain, repository} {
		if _, err := parser.ParseFile(token.NewFileSet(), "generated.go", source, parser.AllErrors); err != nil {
			t.Fatalf("generated invalid activity-log Go file: %v", err)
		}
	}
	for _, fragment := range []string{
		"ActivityLogSubjectType() activitylogdomain.ActivitySubjectType",
		"ActivityLogID()",
		"activity:\"track\"",
	} {
		if !strings.Contains(string(domain), fragment) {
			t.Fatalf("generated domain missing %q: %s", fragment, domain)
		}
	}
	for _, fragment := range []string{
		"activitylogdomain.ActivityLogRepository",
		"postgresinfra.NewActivityLoggingRepository",
	} {
		if !strings.Contains(string(repository), fragment) {
			t.Fatalf("generated repository missing %q: %s", fragment, repository)
		}
	}
}

func TestRuntimeCommandsValidateConfiguration(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("JWT_SECRET", "")
	for _, name := range []string{"serve", "queue:work", "schedule:work", "dev"} {
		if err := Run(context.Background(), []string{name}); err == nil || !strings.Contains(err.Error(), "load configuration") {
			t.Fatalf("%s should reject missing JWT configuration: %v", name, err)
		}
	}
}

func TestExternalTestsRequireServices(t *testing.T) {
	t.Setenv("TEST_DATABASE_URL", "")
	for _, name := range []string{"test", "test:e2e"} {
		if err := Run(context.Background(), []string{name, "--external-services"}); err == nil || !strings.Contains(err.Error(), "TEST_DATABASE_URL") {
			t.Fatalf("%s should reject missing service configuration: %v", name, err)
		}
	}
}

func TestSuperviseStopsChildOnCancel(t *testing.T) {
	ready := filepath.Join(t.TempDir(), "ready")
	child := exec.Command(os.Args[0], "-test.run=^TestSuperviseHelper$")
	child.Env = replaceEnvironment(os.Environ(), map[string]string{"GIN_CLI_TEST_READY": ready})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- supervise(ctx, []*exec.Cmd{child}, time.Second) }()
	deadline := time.After(5 * time.Second)
	for {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		select {
		case err := <-done:
			t.Fatalf("supervisor stopped before child was ready: %v", err)
		case <-deadline:
			t.Fatal("child did not start")
		case <-time.After(10 * time.Millisecond):
		}
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("supervisor did not stop child")
	}
	if _, err := os.Stat(ready + ".stopped"); err != nil {
		t.Fatalf("child did not shut down gracefully: %v", err)
	}
}

func TestSuperviseHelper(t *testing.T) {
	ready := os.Getenv("GIN_CLI_TEST_READY")
	if ready == "" {
		return
	}
	if err := os.WriteFile(ready, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(io.Discard, os.Stdin); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ready+".stopped", nil, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestExitCodePreservesWrappedProcessErrors(t *testing.T) {
	err := errors.New("execution failed")
	if ExitCode(err) != 1 || ExitCode(invalidArguments("bad input")) != 2 {
		t.Fatal("unexpected error exit codes")
	}
}
