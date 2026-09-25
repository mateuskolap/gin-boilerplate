package bootstrap

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gin-boilerplate/config"
	queueinfra "gin-boilerplate/internal/infra/queue"
	"gin-boilerplate/internal/infra/repository"
	"gin-boilerplate/internal/usecase"
	"gin-boilerplate/internal/usecase/jobs"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type WorkerApplication struct {
	SQLDB      *sql.DB
	QueueRedis *redis.Client
	Worker     *queueinfra.Worker
}

func NewWorkerApplication(cfg *config.Config, logger *slog.Logger) (*WorkerApplication, error) {
	db, sqlDB, err := openDatabase(cfg)
	if err != nil {
		return nil, err
	}
	queueRedis, err := openQueueRedis(cfg)
	if err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	maintenanceUseCase := usecase.NewRefreshTokenMaintenanceUseCase(refreshTokenRepo)
	worker, err := queueinfra.NewWorker(
		queueRedis,
		queueinfra.WorkerConfig{
			Concurrency:     cfg.QueueConcurrency,
			ShutdownTimeout: cfg.QueueShutdownTimeout,
		},
		logger,
		jobs.NewPurgeExpiredRefreshTokensHandler(maintenanceUseCase, cfg.RefreshTokenRetention),
	)
	if err != nil {
		_ = queueRedis.Close()
		_ = sqlDB.Close()
		return nil, fmt.Errorf("initialize queue worker: %w", err)
	}
	return &WorkerApplication{SQLDB: sqlDB, QueueRedis: queueRedis, Worker: worker}, nil
}

func (a *WorkerApplication) Start() error {
	return a.Worker.Start()
}

func (a *WorkerApplication) Shutdown() error {
	if a.Worker != nil {
		a.Worker.Shutdown()
	}
	return a.Close()
}

func (a *WorkerApplication) Close() error {
	var databaseError, queueRedisError error
	if a.SQLDB != nil {
		databaseError = a.SQLDB.Close()
	}
	if a.QueueRedis != nil {
		queueRedisError = a.QueueRedis.Close()
	}
	return errors.Join(databaseError, queueRedisError)
}

type SchedulerApplication struct {
	QueueRedis *redis.Client
	Scheduler  *queueinfra.Scheduler
}

func NewSchedulerApplication(cfg *config.Config, logger *slog.Logger) (*SchedulerApplication, error) {
	queueRedis, err := openQueueRedis(cfg)
	if err != nil {
		return nil, err
	}
	scheduler, err := queueinfra.NewScheduler(queueRedis, jobs.NewMaintenanceSchedule(), logger)
	if err != nil {
		_ = queueRedis.Close()
		return nil, fmt.Errorf("initialize queue scheduler: %w", err)
	}
	return &SchedulerApplication{QueueRedis: queueRedis, Scheduler: scheduler}, nil
}

func (a *SchedulerApplication) Start() error {
	return a.Scheduler.Start()
}

func (a *SchedulerApplication) Shutdown() error {
	if a.Scheduler != nil {
		a.Scheduler.Shutdown()
	}
	return a.Close()
}

func (a *SchedulerApplication) Close() error {
	if a.QueueRedis != nil {
		return a.QueueRedis.Close()
	}
	return nil
}

func openDatabase(cfg *config.Config) (*gorm.DB, *sql.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DatabaseConfig.URL()), &gorm.Config{
		TranslateError:       true,
		DisableAutomaticPing: true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("connect to database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("get database instance: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.DBMaxOpenConnections)
	sqlDB.SetMaxIdleConns(cfg.DBMaxIdleConnections)
	sqlDB.SetConnMaxLifetime(cfg.DBConnectionLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.DBConnectionIdleTime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, nil, fmt.Errorf("ping database: %w", err)
	}
	return db, sqlDB, nil
}

func openQueueRedis(cfg *config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.QueueRedisDB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("connect to queue Redis: %w", err)
	}
	return client, nil
}
