package bootstrap

import (
	"context"
	"crypto/tls"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"gin-boilerplate/config"
	activityloghttp "gin-boilerplate/internal/activity_logs/adapters/http"
	activitylogpostgres "gin-boilerplate/internal/activity_logs/adapters/postgres"
	activitylogapp "gin-boilerplate/internal/activity_logs/application"
	deliveryHttp "gin-boilerplate/internal/delivery/http"
	"gin-boilerplate/internal/delivery/http/middleware"
	v1 "gin-boilerplate/internal/delivery/http/v1"
	"gin-boilerplate/internal/domain/port"
	"gin-boilerplate/internal/infra/health"
	imageinfra "gin-boilerplate/internal/infra/image"
	postgresinfra "gin-boilerplate/internal/infra/postgres"
	queueinfra "gin-boilerplate/internal/infra/queue"
	"gin-boilerplate/internal/infra/ratelimit"
	"gin-boilerplate/internal/infra/repository"
	securityinfra "gin-boilerplate/internal/infra/security"
	"gin-boilerplate/internal/infra/seeder"
	"gin-boilerplate/internal/infra/storage"
	organizationhttp "gin-boilerplate/internal/organizations/adapters/http"
	organizationpostgres "gin-boilerplate/internal/organizations/adapters/postgres"
	organizationapp "gin-boilerplate/internal/organizations/application"
	permissionhttp "gin-boilerplate/internal/permissions/adapters/http"
	permissionpostgres "gin-boilerplate/internal/permissions/adapters/postgres"
	permissionapp "gin-boilerplate/internal/permissions/application"
	refreshhttp "gin-boilerplate/internal/refresh_tokens/adapters/http"
	refreshpostgres "gin-boilerplate/internal/refresh_tokens/adapters/postgres"
	refreshapp "gin-boilerplate/internal/refresh_tokens/application"
	rolehttp "gin-boilerplate/internal/roles/adapters/http"
	rolepostgres "gin-boilerplate/internal/roles/adapters/postgres"
	roleapp "gin-boilerplate/internal/roles/application"
	usercache "gin-boilerplate/internal/users/adapters/cache"
	userhttp "gin-boilerplate/internal/users/adapters/http"
	userpostgres "gin-boilerplate/internal/users/adapters/postgres"
	userapp "gin-boilerplate/internal/users/application"

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

	redisClient := redis.NewClient(newRedisOptions(cfg, cfg.RedisDB))

	redisContext, cancelRedis := context.WithTimeout(context.Background(), 5*time.Second)
	if err := redisClient.Ping(redisContext).Err(); err != nil {
		cancelRedis()
		_ = redisClient.Close()
		_ = sqlDB.Close()
		return nil, fmt.Errorf("connect to Redis: %w", err)
	}
	cancelRedis()

	queueRedisClient := redis.NewClient(newRedisOptions(cfg, cfg.QueueRedisDB))
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
	tokenBlacklistRepo := usercache.NewTokenBlackListRepository(cacheRepo)
	activityLogRepo := activitylogpostgres.NewActivityLogRepository(db)
	userRepo := userpostgres.NewUserRepository(db, activityLogRepo)
	roleRepo := rolepostgres.NewRoleRepository(db, activityLogRepo)
	permissionRepo := permissionpostgres.NewPermissionRepository(db)
	refreshTokenRepo := refreshpostgres.NewRefreshTokenRepository(db)
	authorizationRepo := permissionpostgres.NewAuthorizationRepository(db)
	organizationRepo := organizationpostgres.NewOrganizationRepository(db, activityLogRepo)
	txManager := postgresinfra.NewGormTransactionManagerRepository(db)
	rateLimiter := ratelimit.NewRedisLimiter(redisClient)
	passwordChecker := securityinfra.NewPwnedPasswordChecker()

	refreshTokenUseCase := refreshapp.NewRefreshTokenUseCase(refreshTokenRepo, txManager, cfg.RefreshExpiration)
	organizationUseCase := organizationapp.NewOrganizationUseCase(organizationRepo)
	authUseCase := userapp.NewAuthUseCase(
		userRepo,
		roleRepo,
		refreshTokenUseCase,
		organizationUseCase,
		tokenBlacklistRepo,
		txManager,
		cfg.JWTSecret,
		cfg.JWTIssuer,
		cfg.JWTAudience,
		cfg.JWTExpiration,
		cfg.PasswordValidationLevel,
		passwordChecker,
		rateLimiter,
	)
	userUseCase := userapp.NewUserUseCase(
		userRepo,
		refreshTokenUseCase,
		organizationUseCase,
		localStore,
		imageInspector,
		tokenBlacklistRepo,
		txManager,
		cfg.JWTExpiration,
		activityLogRepo,
	)
	roleUseCase := roleapp.NewRoleUseCase(roleRepo, txManager, activityLogRepo)
	permissionUseCase := permissionapp.NewPermissionUseCase(permissionRepo)
	activityLogUseCase := activitylogapp.NewActivityLogUseCase(activityLogRepo)
	permissionCheckerUseCase := permissionapp.NewPermissionCheckerUseCase(authorizationRepo)

	cookieSameSite := http.SameSiteLaxMode
	switch cfg.AuthCookieSameSite {
	case "strict":
		cookieSameSite = http.SameSiteStrictMode
	case "none":
		cookieSameSite = http.SameSiteNoneMode
	}
	router, err := deliveryHttp.SetupRouter(deliveryHttp.RouterConfig{
		AuthHandler:         userhttp.NewAuthHandler(authUseCase, cfg.AuthTokenTransport == "cookie", cfg.Env == "production", cfg.JWTExpiration, cfg.RefreshExpiration, cookieSameSite),
		UserHandler:         userhttp.NewUserHandler(userUseCase),
		RoleHandler:         rolehttp.NewRoleHandler(roleUseCase),
		PermissionHandler:   permissionhttp.NewPermissionHandler(permissionUseCase),
		ActivityLogHandler:  activityloghttp.NewActivityLogHandler(activityLogUseCase),
		RefreshTokenHandler: refreshhttp.NewRefreshTokenHandler(refreshTokenUseCase, cfg.AuthTokenTransport == "cookie"),
		OrganizationHandler: organizationhttp.NewOrganizationHandler(organizationUseCase),
		HealthHandler:       v1.NewHealthHandler(health.NewChecker(sqlDB, redisClient)),
		AuthUseCase:         authUseCase,
		PermissionChecker:   permissionCheckerUseCase,
		RateLimiter:         rateLimiter,
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

func newRedisOptions(cfg *config.Config, db int) *redis.Options {
	options := &redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       db,
	}
	if cfg.Env == "production" {
		options.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12, ServerName: cfg.RedisHost}
	}
	return options
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
