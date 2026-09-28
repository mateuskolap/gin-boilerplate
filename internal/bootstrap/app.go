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
	imageinfra "gin-boilerplate/internal/infra/image"
	queueinfra "gin-boilerplate/internal/infra/queue"
	"gin-boilerplate/internal/infra/ratelimit"
	"gin-boilerplate/internal/infra/repository"
	"gin-boilerplate/internal/infra/seeder"
	"gin-boilerplate/internal/infra/storage"
	"gin-boilerplate/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Application struct {
	Config           *config.Config
	DB               *gorm.DB
	SQLDB            *sql.DB
	RedisClient      *redis.Client
	QueueRedisClient *redis.Client
	Queue            port.QueueDispatcher
	Storage          port.Storage
	Router           *gin.Engine
	Server           *http.Server
	localStore       *storage.Local
}

func NewApplication(cfg *config.Config) (*Application, error) {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	middleware.InitValidator()

	db, sqlDB, err := openDatabase(cfg)
	if err != nil {
		return nil, err
	}

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

	queueRedisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.QueueRedisDB,
	})
	queueRedisContext, cancelQueueRedis := context.WithTimeout(context.Background(), 5*time.Second)
	if err := queueRedisClient.Ping(queueRedisContext).Err(); err != nil {
		cancelQueueRedis()
		_ = queueRedisClient.Close()
		_ = redisClient.Close()
		_ = sqlDB.Close()
		return nil, fmt.Errorf("connect to queue Redis: %w", err)
	}
	cancelQueueRedis()

	localStore, err := storage.NewLocal(cfg.StorageRoot)
	if err != nil {
		_ = queueRedisClient.Close()
		_ = redisClient.Close()
		_ = sqlDB.Close()
		return nil, fmt.Errorf("initialize local storage: %w", err)
	}
	imageInspector := &imageinfra.Inspector{}

	cacheRepo := repository.NewRedisCache(redisClient)
	tokenBlacklistRepo := repository.NewTokenBlackListRepository(cacheRepo)
	activityLogRepo := repository.NewActivityLogRepository(db)
	userRepo := repository.NewUserRepository(db, activityLogRepo)
	roleRepo := repository.NewRoleRepository(db, activityLogRepo)
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
		txManager,
		cfg.JWTSecret,
		cfg.JWTIssuer,
		cfg.JWTAudience,
		cfg.JWTExpiration,
	)
	userUseCase := usecase.NewUserUseCase(
		userRepo,
		refreshTokenUseCase,
		localStore,
		imageInspector,
		tokenBlacklistRepo,
		txManager,
		cfg.JWTExpiration,
		activityLogRepo,
	)
	roleUseCase := usecase.NewRoleUseCase(roleRepo, txManager, activityLogRepo)
	permissionUseCase := usecase.NewPermissionUseCase(permissionRepo)
	activityLogUseCase := usecase.NewActivityLogUseCase(activityLogRepo)
	permissionCheckerUseCase := usecase.NewPermissionCheckerUseCase(authorizationRepo)

	router, err := deliveryHttp.SetupRouter(deliveryHttp.RouterConfig{
		AuthHandler:         v1.NewAuthHandler(authUseCase, cfg.AuthTokenTransport == "cookie", cfg.Env == "production", cfg.JWTExpiration, cfg.RefreshExpiration),
		UserHandler:         v1.NewUserHandler(userUseCase),
		RoleHandler:         v1.NewRoleHandler(roleUseCase),
		PermissionHandler:   v1.NewPermissionHandler(permissionUseCase),
		ActivityLogHandler:  v1.NewActivityLogHandler(activityLogUseCase),
		RefreshTokenHandler: v1.NewRefreshTokenHandler(refreshTokenUseCase, cfg.AuthTokenTransport == "cookie"),
		HealthHandler:       v1.NewHealthHandler(health.NewChecker(sqlDB, redisClient)),
		AuthUseCase:         authUseCase,
		PermissionChecker:   permissionCheckerUseCase,
		RateLimiter:         ratelimit.NewRedisLimiter(redisClient),
		TrustedProxies:      cfg.TrustedProxies,
		CORSAllowedOrigins:  cfg.CORSAllowedOrigins,
		UseAuthCookies:      cfg.AuthTokenTransport == "cookie",
		Env:                 cfg.Env,
	})
	if err != nil {
		_ = localStore.Close()
		_ = queueRedisClient.Close()
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
		Config:           cfg,
		DB:               db,
		SQLDB:            sqlDB,
		RedisClient:      redisClient,
		QueueRedisClient: queueRedisClient,
		Queue:            queueinfra.NewDispatcher(queueRedisClient),
		Storage:          localStore,
		Router:           router,
		Server:           server,
		localStore:       localStore,
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
	var storageError, databaseError, redisError, queueRedisError error
	if a.localStore != nil {
		storageError = a.localStore.Close()
	}
	if a.SQLDB != nil {
		databaseError = a.SQLDB.Close()
	}
	if a.RedisClient != nil {
		redisError = a.RedisClient.Close()
	}
	if a.QueueRedisClient != nil {
		queueRedisError = a.QueueRedisClient.Close()
	}
	return errors.Join(storageError, databaseError, redisError, queueRedisError)
}

func (a *Application) Shutdown(ctx context.Context) error {
	return errors.Join(a.Server.Shutdown(ctx), a.Close())
}
