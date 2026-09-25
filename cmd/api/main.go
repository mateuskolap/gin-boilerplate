package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gin-boilerplate/config"
	"gin-boilerplate/internal/bootstrap"
	"gin-boilerplate/internal/infra/logging"
)

// @title                      Gin Boilerplate API
// @version                    1.0
// @description                Production-ready API boilerplate built with Gin, GORM, and Redis.

// @license.name               MIT
// @license.url                https://opensource.org/licenses/MIT

// @host                       localhost:8080
// @BasePath                   /

// @securityDefinitions.apikey BearerAuth
// @in                         header
// @name                       Authorization
// @description                Type "Bearer" followed by a space and JWT token (e.g. "Bearer eyJhbGci...").
func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() (runErr error) {
	var logFile *os.File
	defer func() {
		if runErr != nil {
			slog.Error("application stopped", "error", runErr)
		}
		if logFile != nil {
			if err := logFile.Close(); err != nil {
				runErr = errors.Join(runErr, fmt.Errorf("close log file: %w", err))
				fmt.Fprintln(os.Stderr, runErr)
			}
		}
	}()

	seedFlag := flag.Bool("seed", false, "Run database seeders and run the application")
	seedOnlyFlag := flag.Bool("seed-only", false, "Run database seeders only")
	flag.Parse()

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	logger, file, err := logging.New(cfg.Env, os.Stdout, logging.FilePath)
	if err != nil {
		return fmt.Errorf("configure logger: %w", err)
	}
	logFile = file
	slog.SetDefault(logger)

	app, err := bootstrap.NewApplication(cfg)
	if err != nil {
		return fmt.Errorf("initialize application: %w", err)
	}

	if *seedFlag || *seedOnlyFlag {
		seedContext, cancelSeed := context.WithTimeout(context.Background(), 30*time.Second)
		err = app.Seed(seedContext)
		cancelSeed()
		if err != nil {
			return errors.Join(fmt.Errorf("run seeders: %w", err), app.Close())
		}
		slog.Info("seeders completed")

		if *seedOnlyFlag {
			return app.Close()
		}
	}

	serverError := make(chan error, 1)
	go func() {
		serverError <- app.Run()
	}()

	signalContext, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	select {
	case err := <-serverError:
		return errors.Join(err, app.Close())
	case <-signalContext.Done():
		slog.Info("shutdown signal received")
	}

	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancelShutdown()
	if err := app.Shutdown(shutdownContext); err != nil {
		return fmt.Errorf("shutdown application: %w", err)
	}

	slog.Info("server stopped")
	return nil
}
