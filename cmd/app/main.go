package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"gin-boilerplate/internal/cli"
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
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// Supervised processes stop on parent pipe closure, including Windows without a console.
	if os.Getenv("GIN_APP_SUPERVISED") == "1" {
		go func() {
			_, _ = io.Copy(io.Discard, os.Stdin)
			cancel()
		}()
	}
	if err := cli.Run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(cli.ExitCode(err))
	}
}
