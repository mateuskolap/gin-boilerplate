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
		slog.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	seedFlag := flag.Bool("seed", false, "Run database seeders and run the application")
	seedOnlyFlag := flag.Bool("seed-only", false, "Run database seeders only")
	flag.Parse()

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	configureLogger(cfg.Env)

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

func configureLogger(environment string) {
	options := &slog.HandlerOptions{Level: slog.LevelInfo}
	var handler slog.Handler = slog.NewTextHandler(os.Stdout, options)
	if environment == "production" {
		handler = slog.NewJSONHandler(os.Stdout, options)
	}
	slog.SetDefault(slog.New(handler))
}
