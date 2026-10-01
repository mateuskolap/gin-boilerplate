package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type command struct {
	name, usage, description string
	minArgs, maxArgs         int
	run                      func(context.Context, []string) error
}

// This is the single catalog used for dispatch and help.
var commands = []command{
	{"serve", "serve", "Start the HTTP API", 0, 0, serve},
	{"queue:work", "queue:work", "Start the queue worker", 0, 0, work},
	{"schedule:work", "schedule:work", "Start the continuous scheduler", 0, 0, schedule},
	{"dev", "dev [--services] [--swagger] [--migrate] [--seed]", "Prepare and supervise API, worker and scheduler", -1, -1, dev},
	{"migrate", "migrate", "Apply pending migrations", 0, 0, func(context.Context, []string) error { return runMigrations("up", 0) }},
	{"migrate:rollback", "migrate:rollback [steps]", "Roll back migrations (default: one step)", 0, 1, rollback},
	{"migrate:version", "migrate:version", "Show the current migration version", 0, 0, func(context.Context, []string) error { return runMigrations("version", 0) }},
	{"db:seed", "db:seed", "Run transactional database seeders", 0, 0, seed},
	{"make:domain", "make:domain Domain", "Generate a feature's domain, application and PostgreSQL adapter", 1, 1, generateDomain},
	{"make:migration", "make:migration name", "Create paired up/down SQL migration files", 1, 1, generateMigration},
	{"docs:generate", "docs:generate", "Regenerate Swagger documentation", 0, 0, generateDocs},
	{"test", "test [--external-services]", "Run all tests with disposable PostgreSQL and Redis", -1, -1, testAll},
	{"test:e2e", "test:e2e [--external-services]", "Run authentication HTTP E2E tests", -1, -1, testE2E},
	{"vulncheck", "vulncheck", "Check known vulnerabilities with the versioned Go tool", 0, 0, func(ctx context.Context, _ []string) error { return runTool(ctx, "go", "tool", "govulncheck", "./...") }},
}

type argumentError struct{ error }

func invalidArguments(format string, args ...any) error {
	return argumentError{fmt.Errorf(format, args...)}
}

func ExitCode(err error) int {
	var args argumentError
	if errors.As(err, &args) {
		return 2
	}
	var process *exec.ExitError
	if errors.As(err, &process) && process.ExitCode() > 0 {
		return process.ExitCode()
	}
	return 1
}

func Run(ctx context.Context, args []string) error {
	if len(args) == 0 || (len(args) == 1 && (args[0] == "help" || args[0] == "--help" || args[0] == "-h")) {
		fmt.Println("Usage: app <command> [arguments]\n\nCommands:")
		for _, command := range commands {
			fmt.Printf("  %-20s %s\n", command.name, command.description)
		}
		return nil
	}
	if args[0] == "help" {
		if len(args) != 2 {
			return invalidArguments("usage: app help [command]")
		}
		args = []string{args[1], "--help"}
	}
	for _, command := range commands {
		if command.name != args[0] {
			continue
		}
		if len(args) == 2 && (args[1] == "--help" || args[1] == "-h") {
			fmt.Printf("Usage: app %s\n\n%s\n", command.usage, command.description)
			return nil
		}
		values := args[1:]
		if command.minArgs >= 0 && (len(values) < command.minArgs || len(values) > command.maxArgs) {
			return invalidArguments("usage: app %s", command.usage)
		}
		return command.run(ctx, values)
	}
	return invalidArguments("unknown command %q; run app help", args[0])
}

func parseFlags(flags *flag.FlagSet, args []string) error {
	if err := flags.Parse(args); err != nil {
		return argumentError{err}
	}
	if flags.NArg() != 0 {
		return invalidArguments("unexpected arguments: %v", flags.Args())
	}
	return nil
}

func rollback(_ context.Context, args []string) error {
	steps := 1
	if len(args) == 1 {
		var err error
		steps, err = strconv.Atoi(args[0])
		if err != nil || steps < 1 {
			return invalidArguments("rollback step count must be a positive integer")
		}
	}
	return runMigrations("down", steps)
}

func generateDomain(_ context.Context, args []string) error {
	if _, _, _, _, err := domainNames(args[0]); err != nil {
		return argumentError{err}
	}
	folder, err := makeDomain(".", args[0])
	if err == nil {
		fmt.Printf("Created internal/%s\n", folder)
	}
	return err
}

func generateMigration(_ context.Context, args []string) error {
	name, err := normalizeName(args[0])
	if err != nil {
		return argumentError{err}
	}
	return createMigration(migrationsDirectory, time.Now().UTC(), name)
}

func generateDocs(ctx context.Context, _ []string) error {
	features, err := filepath.Glob(filepath.Join("internal", "*", "adapters", "http"))
	if err != nil {
		return err
	}
	directories := []string{"./cmd/app", "./internal/delivery/http"}
	for _, feature := range features {
		directories = append(directories, "./"+filepath.ToSlash(feature))
	}
	return runTool(ctx, "go", "tool", "swag", "init", "-d", strings.Join(directories, ","), "-g", "main.go", "-o", "docs", "--parseInternal")
}

func developmentCompose(ctx context.Context, args ...string) error {
	return runTool(ctx, "docker", append([]string{"compose", "--file", "docker-compose.yaml"}, args...)...)
}

func runTool(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s failed: %w", name, err)
	}
	return nil
}
