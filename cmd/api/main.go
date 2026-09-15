package main

import (
	"context"
	"flag"
	"log"
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
	seedFlag := flag.Bool("seed", false, "Run database seeders and run the application")
	seedOnlyFlag := flag.Bool("seed-only", false, "Run database seeders only")
	flag.Parse()

	cfg := config.LoadConfig()

	app, err := bootstrap.NewApplication(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	if *seedFlag || *seedOnlyFlag {
		log.Println("Running seeders...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := app.Seed(ctx); err != nil {
			log.Fatalf("Failed to run seeders: %v", err)
		}

		log.Println("Seeders executed successfully!")

		if *seedOnlyFlag {
			_ = app.Close()
			return
		}
	}

	go func() {
		if err := app.Run(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited successfully")
}
