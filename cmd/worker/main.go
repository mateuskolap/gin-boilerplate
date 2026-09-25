package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"gin-boilerplate/config"
	"gin-boilerplate/internal/bootstrap"
	"gin-boilerplate/internal/infra/logging"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() (runErr error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	logger, logFile, err := logging.New(cfg.Env, os.Stdout, logging.QueueFilePath)
	if err != nil {
		return fmt.Errorf("configure queue logger: %w", err)
	}
	defer func() {
		if err := logFile.Close(); err != nil {
			runErr = errors.Join(runErr, fmt.Errorf("close queue log file: %w", err))
		}
	}()
	slog.SetDefault(logger)

	app, err := bootstrap.NewWorkerApplication(cfg, logger)
	if err != nil {
		return fmt.Errorf("initialize queue worker: %w", err)
	}
	if err := app.Start(); err != nil {
		_ = app.Close()
		return fmt.Errorf("start queue worker: %w", err)
	}
	slog.Info("queue worker started", "concurrency", cfg.QueueConcurrency)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	slog.Info("queue worker shutdown signal received")
	if err := app.Shutdown(); err != nil {
		return fmt.Errorf("shutdown queue worker: %w", err)
	}
	slog.Info("queue worker stopped")
	return nil
}
