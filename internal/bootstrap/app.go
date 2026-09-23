package bootstrap

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"gin-boilerplate/config"
	deliveryHttp "gin-boilerplate/internal/delivery/http"
	"gin-boilerplate/internal/delivery/http/middleware"
	v1 "gin-boilerplate/internal/delivery/http/v1"
	"gin-boilerplate/internal/domain/port"
	"gin-boilerplate/internal/infra/health"
	"gin-boilerplate/internal/infra/ratelimit"
	"gin-boilerplate/internal/infra/repository"
	"gin-boilerplate/internal/infra/seeder"
	"gin-boilerplate/internal/infra/storage"
	"gin-boilerplate/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Application struct {
	Config      *config.Config
	DB          *gorm.DB
	SQLDB       *sql.DB
	RedisClient *redis.Client
	Storage     port.Storage
	Router      *gin.Engine
	Server      *http.Server
	localStore  *storage.Local
}

func NewApplication(cfg *config.Config) (*Application, error) {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	middleware.InitValidator()

	db, err := gorm.Open(postgres.Open(cfg.DatabaseConfig.URL()), &gorm.Config{
		TranslateError:       true,
		DisableAutomaticPing: true,
	})
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get database instance: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.DBMaxOpenConnections)
	sqlDB.SetMaxIdleConns(cfg.DBMaxIdleConnections)
	sqlDB.SetConnMaxLifetime(cfg.DBConnectionLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.DBConnectionIdleTime)

	databaseContext, cancelDatabase := context.WithTimeout(context.Background(), 5*time.Second)
	if err := sqlDB.PingContext(databaseContext); err != nil {
		cancelDatabase()
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	cancelDatabase()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	redisContext, cancelRedis := context.WithTimeout(context.Background(), 5*time.Second)
	if err := redisClient.Ping(redisContext).Err(); err != nil {
		cancelRedis()
		_ = redisClient.Close()
		_ = sqlDB.Close()
		return nil, fmt.Errorf("connect to Redis: %w", err)
	}
	cancelRedis()

	localStore, err := storage.NewLocal(cfg.StorageRoot, cfg.StorageMaxFileSize)
	if err != nil {
		_ = redisClient.Close()
		_ = sqlDB.Close()
		return nil, fmt.Errorf("initialize local storage: %w", err)
	}

	cacheRepo := repository.NewRedisCache(redisClient)
	tokenBlacklistRepo := repository.NewTokenBlackListRepository(cacheRepo)
	userRepo := repository.NewUserRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	permissionRepo := repository.NewPermissionRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	authorizationRepo := repository.NewAuthorizationRepository(db)
	txManager := repository.NewGormTransactionManagerRepository(db)

	refreshTokenUseCase := usecase.NewRefreshTokenUseCase(refreshTokenRepo, txManager, cfg.RefreshExpiration)
	authUseCase := usecase.NewAuthUseCase(
		userRepo,
		roleRepo,
		refreshTokenUseCase,
		tokenBlacklistRepo,
		cfg.JWTSecret,
		cfg.JWTIssuer,
		cfg.JWTAudience,
		cfg.JWTExpiration,
	)
	userUseCase := usecase.NewUserUseCase(
		userRepo,
		refreshTokenUseCase,
		tokenBlacklistRepo,
		txManager,
		cfg.JWTExpiration,
	)
	roleUseCase := usecase.NewRoleUseCase(roleRepo)
	permissionUseCase := usecase.NewPermissionUseCase(permissionRepo)
	permissionCheckerUseCase := usecase.NewPermissionCheckerUseCase(authorizationRepo)

	router, err := deliveryHttp.SetupRouter(deliveryHttp.RouterConfig{
		AuthHandler:         v1.NewAuthHandler(authUseCase),
		UserHandler:         v1.NewUserHandler(userUseCase),
		RoleHandler:         v1.NewRoleHandler(roleUseCase),
		PermissionHandler:   v1.NewPermissionHandler(permissionUseCase),
		RefreshTokenHandler: v1.NewRefreshTokenHandler(refreshTokenUseCase),
		HealthHandler:       v1.NewHealthHandler(health.NewChecker(sqlDB, redisClient)),
		AuthUseCase:         authUseCase,
		PermissionChecker:   permissionCheckerUseCase,
		RateLimiter:         ratelimit.NewRedisLimiter(redisClient),
		TrustedProxies:      cfg.TrustedProxies,
		CORSAllowedOrigins:  cfg.CORSAllowedOrigins,
		Env:                 cfg.Env,
	})
	if err != nil {
		_ = localStore.Close()
		_ = redisClient.Close()
		_ = sqlDB.Close()
		return nil, err
	}

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           router,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	return &Application{
		Config:      cfg,
		DB:          db,
		SQLDB:       sqlDB,
		RedisClient: redisClient,
		Storage:     localStore,
		Router:      router,
		Server:      server,
		localStore:  localStore,
	}, nil
}

func (a *Application) Run() error {
	slog.Info("server started", "port", a.Config.Port)
	if err := a.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (a *Application) Seed(ctx context.Context) error {
	if err := a.Config.ValidateSeeder(); err != nil {
		return err
	}
	return seeder.NewDatabaseSeeder(a.DB, a.Config).Run(ctx)
}

func (a *Application) Close() error {
	var storageError, databaseError, redisError error
	if a.localStore != nil {
		storageError = a.localStore.Close()
	}
	if a.SQLDB != nil {
		databaseError = a.SQLDB.Close()
	}
	if a.RedisClient != nil {
		redisError = a.RedisClient.Close()
	}
	return errors.Join(storageError, databaseError, redisError)
}

func (a *Application) Shutdown(ctx context.Context) error {
	return errors.Join(a.Server.Shutdown(ctx), a.Close())
}
