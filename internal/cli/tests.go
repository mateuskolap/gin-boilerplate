package cli

import (
	"context"
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func testAll(ctx context.Context, args []string) error { return runTests(ctx, args, false) }
func testE2E(ctx context.Context, args []string) error { return runTests(ctx, args, true) }

func runTests(ctx context.Context, args []string, e2e bool) (resultErr error) {
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	external := flags.Bool("external-services", false, "Use existing services configured by TEST_* variables")
	if err := parseFlags(flags, args); err != nil {
		return err
	}
	environment := os.Environ()
	if *external {
		for _, key := range []string{"TEST_DATABASE_URL", "TEST_REDIS_ADDR", "TEST_REDIS_DB", "TEST_QUEUE_REDIS_DB"} {
			if strings.TrimSpace(os.Getenv(key)) == "" {
				return fmt.Errorf("--external-services requires %s", key)
			}
		}
	} else {
		project := "gin-boilerplate-test-" + strings.ToLower(rand.Text())
		base := []string{"compose", "--project-name", project, "--file", "docker-compose.test.yaml"}
		// Cleanup uses its own context, including when startup or tests are cancelled.
		defer func() {
			cleanup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			resultErr = errors.Join(resultErr, runTool(cleanup, "docker", append(base, "down", "--volumes", "--remove-orphans")...))
		}()
		if err := runTool(ctx, "docker", append(base, "up", "--detach", "--wait", "--wait-timeout", "90")...); err != nil {
			return err
		}
		postgresPort, err := composePort(ctx, base, "postgres", "5432")
		if err != nil {
			return err
		}
		redisPort, err := composePort(ctx, base, "redis", "6379")
		if err != nil {
			return err
		}
		environment = replaceEnvironment(environment, map[string]string{
			"TEST_DATABASE_URL":   "postgres://postgres:postgres@127.0.0.1:" + postgresPort + "/test?sslmode=disable",
			"TEST_REDIS_ADDR":     "127.0.0.1:" + redisPort,
			"TEST_REDIS_DB":       "14",
			"TEST_QUEUE_REDIS_DB": "15",
		})
	}
	arguments := []string{"test", "./...", "-count=1", "-cover"}
	if e2e {
		arguments = []string{"test", "./internal/integration", "-run", "^TestAuthenticationE2E$", "-count=1", "-v"}
	}
	cmd := exec.CommandContext(ctx, "go", arguments...)
	cmd.Env = environment
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tests failed: %w", err)
	}
	return nil
}

func composePort(ctx context.Context, base []string, service, target string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", append(base, "port", service, target)...)
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("read %s test port: %w", service, err)
	}
	line := strings.Split(strings.TrimSpace(string(output)), "\n")[0]
	_, port, err := net.SplitHostPort(line)
	if err != nil {
		return "", fmt.Errorf("invalid %s port address %q: %w", service, line, err)
	}
	number, err := strconv.Atoi(port)
	if err != nil || number < 1 || number > 65535 {
		return "", fmt.Errorf("invalid %s test port %q", service, port)
	}
	return port, nil
}

func replaceEnvironment(environment []string, replacements map[string]string) []string {
	result := make([]string, 0, len(environment)+len(replacements))
	for _, value := range environment {
		key, _, _ := strings.Cut(value, "=")
		if _, replaced := replacements[key]; !replaced {
			result = append(result, value)
		}
	}
	for key, value := range replacements {
		result = append(result, key+"="+value)
	}
	return result
}
