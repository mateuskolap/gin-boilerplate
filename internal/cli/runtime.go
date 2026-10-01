package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"gin-boilerplate/config"
	"gin-boilerplate/internal/bootstrap"
	"gin-boilerplate/internal/infra/logging"
)

func runtimeConfig(queue bool) (*config.Config, *os.File, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("load configuration: %w", err)
	}
	path := logging.FilePath
	if queue {
		path = logging.QueueFilePath
	}
	logger, file, err := logging.New(cfg.Env, os.Stdout, path)
	if err != nil {
		return nil, nil, fmt.Errorf("configure logger: %w", err)
	}
	slog.SetDefault(logger)
	return cfg, file, nil
}

func serve(ctx context.Context, _ []string) (resultErr error) {
	cfg, file, err := runtimeConfig(false)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, file.Close()) }()
	app, err := bootstrap.NewApplication(cfg)
	if err != nil {
		return fmt.Errorf("initialize application: %w", err)
	}
	serverError := make(chan error, 1)
	go func() { serverError <- app.Run() }()
	select {
	case err := <-serverError:
		return errors.Join(err, app.Close())
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	}
	shutdownContext, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	return app.Shutdown(shutdownContext)
}

func work(ctx context.Context, _ []string) (resultErr error) {
	cfg, file, err := runtimeConfig(true)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, file.Close()) }()
	app, err := bootstrap.NewWorkerApplication(cfg, slog.Default())
	if err != nil {
		return fmt.Errorf("initialize queue worker: %w", err)
	}
	if err := app.Start(); err != nil {
		return errors.Join(err, app.Close())
	}
	slog.Info("queue worker started", "concurrency", cfg.QueueConcurrency)
	<-ctx.Done()
	return app.Shutdown()
}

func schedule(ctx context.Context, _ []string) (resultErr error) {
	cfg, file, err := runtimeConfig(true)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, file.Close()) }()
	app, err := bootstrap.NewSchedulerApplication(cfg, slog.Default())
	if err != nil {
		return fmt.Errorf("initialize queue scheduler: %w", err)
	}
	if err := app.Start(); err != nil {
		return errors.Join(err, app.Close())
	}
	slog.Info("queue scheduler started", "timezone", "UTC")
	<-ctx.Done()
	return app.Shutdown()
}

func seed(ctx context.Context, _ []string) error {
	cfg, err := config.LoadSeederConfig()
	if err != nil {
		return fmt.Errorf("load seeder configuration: %w", err)
	}
	return bootstrap.RunSeed(ctx, cfg)
}
