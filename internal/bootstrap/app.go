package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"gin-boilerplate/config"
	deliveryHttp "gin-boilerplate/internal/delivery/http"
	"gin-boilerplate/internal/delivery/http/middleware"
	v1 "gin-boilerplate/internal/delivery/http/v1"
	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/infra/repository"
	"gin-boilerplate/internal/infra/seeder"
	"gin-boilerplate/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Application struct {
	Config             *config.Config
	DB                 *gorm.DB
	SQLDB              *sql.DB
	RedisClient        *redis.Client
	Router             *gin.Engine
	Server             *http.Server
	PermissionUseCase  domain.PermissionUseCase
	RolePermissionRepo domain.RolePermissionRepository
}

func NewApplication(cfg *config.Config) (*Application, error) {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	middleware.InitValidator()

	// GORM & Postgres
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort, cfg.DBSSLMode,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	if cfg.Env == "development" {
		if err := db.AutoMigrate(
			&domain.User{},
			&domain.Role{},
			&domain.Permission{},
			&domain.RefreshToken{},
		); err != nil {
			return nil, fmt.Errorf("failed to run database migrations: %w", err)
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
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Repositories
	cacheRepo := repository.NewRedisCache(redisClient)
	tokenBlacklistRepo := repository.NewTokenBlackListRepository(cacheRepo)
	userRepo := repository.NewUserRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	permissionRepo := repository.NewPermissionRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	rolePermissionRepo := repository.NewRolePermissionRepository(cacheRepo)
	txManager := repository.NewGormTransactionManagerRepository(db)

	// UseCases
	refreshTokenUseCase := usecase.NewRefreshTokenUseCase(refreshTokenRepo, txManager, cfg.RefreshExpiration)
	authUseCase := usecase.NewAuthUseCase(userRepo, roleRepo, refreshTokenUseCase, tokenBlacklistRepo, cfg.JWTSecret, cfg.JWTExpiration)
	userUseCase := usecase.NewUserUseCase(userRepo, roleRepo, refreshTokenUseCase, tokenBlacklistRepo, txManager, cfg.JWTExpiration)
	roleUseCase := usecase.NewRoleUseCase(roleRepo, rolePermissionRepo, cfg.RolePermissionsTTL)
	permissionUseCase := usecase.NewPermissionUseCase(permissionRepo, rolePermissionRepo)
	permissionCheckerUseCase := usecase.NewPermissionCheckerUseCase(rolePermissionRepo, roleRepo, cfg.RolePermissionsTTL)

	// Handlers
	authHandler := v1.NewAuthHandler(authUseCase)
	userHandler := v1.NewUserHandler(userUseCase)
	roleHandler := v1.NewRoleHandler(roleUseCase)
	permissionHandler := v1.NewPermissionHandler(permissionUseCase)
	refreshTokenHandler := v1.NewRefreshTokenHandler(refreshTokenUseCase)

	// Router
	router := deliveryHttp.SetupRouter(deliveryHttp.RouterConfig{
		AuthHandler:         authHandler,
		UserHandler:         userHandler,
		RoleHandler:         roleHandler,
		PermissionHandler:   permissionHandler,
		RefreshTokenHandler: refreshTokenHandler,
		AuthUseCase:         authUseCase,
		PermissionChecker:   permissionCheckerUseCase,
		RedisClient:         redisClient,
		Env:                 cfg.Env,
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return &Application{
		Config:             cfg,
		DB:                 db,
		SQLDB:              sqlDB,
		RedisClient:        redisClient,
		Router:             router,
		Server:             srv,
		PermissionUseCase:  permissionUseCase,
		RolePermissionRepo: rolePermissionRepo,
	}, nil
}

func (a *Application) Run() error {
	log.Printf("Server is running on port %s", a.Config.Port)
	if err := a.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (a *Application) Seed(ctx context.Context) error {
	seederRunner := seeder.NewDatabaseSeeder(a.DB, a.Config, a.RolePermissionRepo)
	return seederRunner.Run(ctx)
}

func (a *Application) Close() error {
	var errSql, errRedis error
	if a.SQLDB != nil {
		errSql = a.SQLDB.Close()
	}
	if a.RedisClient != nil {
		errRedis = a.RedisClient.Close()
	}
	if errSql != nil {
		return errSql
	}
	return errRedis
}

func (a *Application) Shutdown(ctx context.Context) error {
	if err := a.Server.Shutdown(ctx); err != nil {
		return err
	}
	return a.Close()
}
