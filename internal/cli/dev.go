package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"gin-boilerplate/config"
)

func dev(ctx context.Context, args []string) (resultErr error) {
	flags := flag.NewFlagSet("dev", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	services := flags.Bool("services", false, "Start development PostgreSQL and Redis")
	swagger := flags.Bool("swagger", false, "Generate Swagger before compiling")
	migrate := flags.Bool("migrate", false, "Apply migrations before starting")
	seeders := flags.Bool("seed", false, "Run seeders before starting")
	if err := parseFlags(flags, args); err != nil {
		return err
	}
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	if *seeders {
		if _, err := config.LoadSeederConfig(); err != nil {
			return err
		}
	}
	if *services {
		if err := developmentCompose(ctx, "up", "--detach", "--wait"); err != nil {
			return err
		}
	}
	if *swagger {
		if err := generateDocs(ctx, nil); err != nil {
			return err
		}
	}
	if *migrate {
		if err := runMigrations("up", 0); err != nil {
			return err
		}
	}
	if *seeders {
		if err := seed(ctx, nil); err != nil {
			return err
		}
	}

	directory, err := os.MkdirTemp("", "gin-boilerplate-dev-")
	if err != nil {
		return err
	}
	binary := filepath.Join(directory, "app")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	defer func() {
		if err := os.Remove(binary); err != nil && !errors.Is(err, os.ErrNotExist) {
			resultErr = errors.Join(resultErr, err)
		}
		resultErr = errors.Join(resultErr, os.Remove(directory))
	}()
	// Rebuild after docs generation so serve uses the newly generated specification.
	if err := runTool(ctx, "go", "build", "-o", binary, "./cmd/app"); err != nil {
		return err
	}
	processes := []*exec.Cmd{
		exec.Command(binary, "serve"),
		exec.Command(binary, "queue:work"),
		exec.Command(binary, "schedule:work"),
	}
	return supervise(ctx, processes, max(cfg.ShutdownTimeout, cfg.QueueShutdownTimeout)+5*time.Second)
}

type processExit struct {
	index int
	err   error
}

type supervisedProcess struct {
	command *exec.Cmd
	stop    *os.File
}

func supervise(ctx context.Context, processes []*exec.Cmd, grace time.Duration) (resultErr error) {
	exits := make(chan processExit, len(processes))
	running := make(map[int]supervisedProcess)
	defer func() {
		for _, process := range running {
			_ = process.stop.Close()
		}
		timer := time.NewTimer(grace)
		defer timer.Stop()
		for len(running) > 0 {
			select {
			case exit := <-exits:
				delete(running, exit.index)
			case <-timer.C:
				resultErr = errors.Join(resultErr, errors.New("development processes exceeded shutdown deadline"))
				for _, process := range running {
					// ponytail: Kill stops direct children only; add process-tree handling if descendants leak.
					_ = process.command.Process.Kill()
				}
			}
		}
	}()
	for index, process := range processes {
		if ctx.Err() != nil {
			return nil
		}
		reader, writer, err := os.Pipe()
		if err != nil {
			return err
		}
		process.Stdin = reader
		process.Env = replaceEnvironment(process.Environ(), map[string]string{"GIN_APP_SUPERVISED": "1"})
		process.Stdout, process.Stderr = os.Stdout, os.Stderr
		if err := process.Start(); err != nil {
			_ = reader.Close()
			_ = writer.Close()
			return fmt.Errorf("start %v: %w", process.Args, err)
		}
		_ = reader.Close()
		running[index] = supervisedProcess{process, writer}
		go func() { exits <- processExit{index, process.Wait()} }()
	}
	select {
	case <-ctx.Done():
		return nil
	case exit := <-exits:
		process := running[exit.index]
		_ = process.stop.Close()
		delete(running, exit.index)
		if exit.err != nil {
			return fmt.Errorf("%v exited unexpectedly: %w", process.command.Args, exit.err)
		}
		return fmt.Errorf("%v exited unexpectedly", process.command.Args)
	}
}
