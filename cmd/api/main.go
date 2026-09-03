package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gin-boilerplate/config"
	deliveryHttp "gin-boilerplate/internal/delivery/http"
	v1 "gin-boilerplate/internal/delivery/http/v1"
	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/repository"
	"gin-boilerplate/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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

	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// GORM & Postgres
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort, cfg.DBSSLMode,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database instance: %v", err)
	}

	if cfg.Env == "development" {
		if err := db.AutoMigrate(
			&domain.User{},
			&domain.Role{},
			&domain.Permission{},
			&domain.RefreshToken{},
		); err != nil {
			log.Fatalf("Failed to run database migrations: %v", err)
		}
	}

	// Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	ctxTimeout, cancelRedis := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelRedis()
	if err := redisClient.Ping(ctxTimeout).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Repositories
	cacheRepo := repository.NewRedisCache(redisClient)
	tokenBlacklistRepo := repository.NewTokenBlackListRepository(cacheRepo)
	userRepo := repository.NewUserRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	permissionRepo := repository.NewPermissionRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)

	// UseCases
	refreshTokenUseCase := usecase.NewRefreshTokenUseCase(refreshTokenRepo, time.Hour*24*7)
	userUseCase := usecase.NewUserUseCase(userRepo, refreshTokenUseCase, tokenBlacklistRepo, cfg.JWTSecret, cfg.JWTExpiration)
	roleUseCase := usecase.NewRoleUseCase(roleRepo)
	permissionUseCase := usecase.NewPermissionUseCase(permissionRepo)

	// Handlers & Router
	authHandler := v1.NewAuthHandler(userUseCase)
	userHandler := v1.NewUserHandler(userUseCase)
	roleHandler := v1.NewRoleHandler(roleUseCase)
	permissionHandler := v1.NewPermissionHandler(permissionUseCase)

	router := deliveryHttp.SetupRouter(deliveryHttp.RouterConfig{
		AuthHandler:       authHandler,
		UserHandler:       userHandler,
		RoleHandler:       roleHandler,
		PermissionHandler: permissionHandler,
		TokenBlacklist:    tokenBlacklistRepo,
		JWTSecret:         cfg.JWTSecret,
		Env:               cfg.Env,
	})

	// Seeders
	if *seedFlag {
		log.Println("Running seeders...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := permissionUseCase.SeedPermissions(ctx); err != nil {
			log.Fatalf("Failed to seed permissions: %v", err)
		}

		log.Println("Seeders executed successfully!")

		if *seedOnlyFlag {
			return
		}
	}

	// Server config
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("Server is running on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Closing database connections...")
	if err := sqlDB.Close(); err != nil {
		log.Printf("Error closing database connection: %v", err)
	}

	log.Println("Closing Redis connection...")
	if err := redisClient.Close(); err != nil {
		log.Printf("Error closing Redis connection: %v", err)
	}

	log.Println("Server exited successfully")
}
