package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gin-boilerplate/internal/domain/port"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

type dispatcher struct {
	client *asynq.Client
}

func NewDispatcher(redisClient redis.UniversalClient) port.QueueDispatcher {
	return &dispatcher{client: asynq.NewClientFromRedisClient(redisClient)}
}

func (d *dispatcher) Dispatch(ctx context.Context, task port.QueueTask, options port.DispatchOptions) (port.DispatchInfo, error) {
	if err := task.Validate(); err != nil {
		return port.DispatchInfo{}, err
	}
	options = normalizeOptions(options)
	if err := options.Validate(); err != nil {
		return port.DispatchInfo{}, err
	}

	asynqTask, asynqOptions := toAsynqTask(task, options)
	info, err := d.client.EnqueueContext(ctx, asynqTask, asynqOptions...)
	if err != nil {
		return port.DispatchInfo{}, fmt.Errorf("enqueue task %q: %w", task.Type, err)
	}
	return dispatchInfo(info), nil
}

func normalizeOptions(options port.DispatchOptions) port.DispatchOptions {
	if options.Queue == "" {
		options.Queue = port.DefaultQueue
	}
	return options
}

func toAsynqTask(task port.QueueTask, options port.DispatchOptions) (*asynq.Task, []asynq.Option) {
	opts := []asynq.Option{
		asynq.Queue(options.Queue),
		asynq.Timeout(options.Timeout),
		asynq.MaxRetry(options.MaxRetries),
	}
	if options.ProcessAt != nil {
		opts = append(opts, asynq.ProcessAt(options.ProcessAt.UTC()))
	}
	if options.UniqueFor > 0 {
		opts = append(opts, asynq.Unique(options.UniqueFor))
	}
	if options.Retention > 0 {
		opts = append(opts, asynq.Retention(options.Retention))
	}
	return asynq.NewTask(task.Type, task.Payload), opts
}

func dispatchInfo(info *asynq.TaskInfo) port.DispatchInfo {
	return port.DispatchInfo{
		ID:        info.ID,
		Queue:     info.Queue,
		Type:      info.Type,
		ProcessAt: info.NextProcessAt,
	}
}

type Worker struct {
	server *asynq.Server
	mux    *asynq.ServeMux
}

type WorkerConfig struct {
	Concurrency     int
	ShutdownTimeout time.Duration
}

func NewWorker(redisClient redis.UniversalClient, config WorkerConfig, logger *slog.Logger, handlers ...port.TaskHandler) (*Worker, error) {
	mux := asynq.NewServeMux()
	mux.Use(func(next asynq.Handler) asynq.Handler {
		return asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) error {
			if !json.Valid(task.Payload()) {
				return fmt.Errorf("%w: task %q has invalid JSON payload", asynq.SkipRetry, task.Type())
			}
			err := next.ProcessTask(ctx, task)
			if errors.Is(err, asynq.ErrHandlerNotFound) {
				return fmt.Errorf("%w: %w", asynq.SkipRetry, err)
			}
			return err
		})
	})
	seen := make(map[string]struct{}, len(handlers))
	for _, handler := range handlers {
		if handler == nil || handler.TaskType() == "" {
			return nil, fmt.Errorf("queue task handler is invalid")
		}
		if _, exists := seen[handler.TaskType()]; exists {
			return nil, fmt.Errorf("duplicate queue task handler %q", handler.TaskType())
		}
		seen[handler.TaskType()] = struct{}{}
		mux.Handle(handler.TaskType(), asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) error {
			return handler.HandleTask(ctx, json.RawMessage(task.Payload()))
		}))
	}

	server := asynq.NewServerFromRedisClient(redisClient, asynq.Config{
		Concurrency:     config.Concurrency,
		Queues:          map[string]int{port.DefaultQueue: 5, "maintenance": 1},
		ShutdownTimeout: config.ShutdownTimeout,
		HealthCheckFunc: func(err error) {
			if err != nil {
				logger.Error("queue Redis health check failed", "error", err)
			}
		},
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
			retried, _ := asynq.GetRetryCount(ctx)
			maxRetries, _ := asynq.GetMaxRetry(ctx)
			queueName, _ := asynq.GetQueueName(ctx)
			taskID, _ := asynq.GetTaskID(ctx)
			message := "queue task failed"
			if retried >= maxRetries || errors.Is(err, asynq.SkipRetry) {
				message = "queue task archived"
			} else {
				message = "queue task will retry"
			}
			logger.ErrorContext(ctx, message,
				"task_id", taskID,
				"task_type", task.Type(),
				"queue", queueName,
				"retry", retried,
				"max_retries", maxRetries,
				"error", err,
			)
		}),
		Logger:   queueLogger{logger: logger},
		LogLevel: asynq.WarnLevel,
	})
	return &Worker{server: server, mux: mux}, nil
}

func (w *Worker) Start() error {
	return w.server.Start(w.mux)
}

func (w *Worker) Shutdown() {
	w.server.Shutdown()
}

type Scheduler struct {
	scheduler *asynq.Scheduler
}

func NewScheduler(redisClient redis.UniversalClient, periodicTasks []port.PeriodicTask, logger *slog.Logger) (*Scheduler, error) {
	s := asynq.NewSchedulerFromRedisClient(redisClient, &asynq.SchedulerOpts{
		Location: time.UTC,
		Logger:   queueLogger{logger: logger},
		LogLevel: asynq.WarnLevel,
		PostEnqueueFunc: func(_ *asynq.TaskInfo, err error) {
			if err != nil {
				logger.Error("scheduled queue task enqueue failed", "error", err)
			}
		},
	})
	for _, periodicTask := range periodicTasks {
		if periodicTask.Name == "" || periodicTask.Cron == "" {
			return nil, fmt.Errorf("periodic task definition is invalid")
		}
		if err := periodicTask.Task.Validate(); err != nil {
			return nil, fmt.Errorf("validate periodic task %q: %w", periodicTask.Name, err)
		}
		options := normalizeOptions(periodicTask.Options)
		if err := options.Validate(); err != nil {
			return nil, fmt.Errorf("validate periodic task %q options: %w", periodicTask.Name, err)
		}
		task, asynqOptions := toAsynqTask(periodicTask.Task, options)
		if _, err := s.Register(periodicTask.Cron, task, asynqOptions...); err != nil {
			return nil, fmt.Errorf("register periodic task %q: %w", periodicTask.Name, err)
		}
	}
	return &Scheduler{scheduler: s}, nil
}

func (s *Scheduler) Start() error {
	return s.scheduler.Start()
}

func (s *Scheduler) Shutdown() {
	s.scheduler.Shutdown()
}

type queueLogger struct {
	logger *slog.Logger
}

func (l queueLogger) Debug(args ...interface{}) {
	l.logger.Debug("asynq", "message", fmt.Sprint(args...))
}
func (l queueLogger) Info(args ...interface{}) {
	l.logger.Info("asynq", "message", fmt.Sprint(args...))
}
func (l queueLogger) Warn(args ...interface{}) {
	l.logger.Warn("asynq", "message", fmt.Sprint(args...))
}
func (l queueLogger) Error(args ...interface{}) {
	l.logger.Error("asynq", "message", fmt.Sprint(args...))
}
func (l queueLogger) Fatal(args ...interface{}) {
	l.logger.Error("asynq fatal", "message", fmt.Sprint(args...))
}
