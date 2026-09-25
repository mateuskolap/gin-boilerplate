package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"gin-boilerplate/internal/domain/port"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

const retryPolicyHeader = "gin-boilerplate.retry-policy"

type retryPolicyWire struct {
	Backoff      port.RetryBackoff `json:"backoff"`
	InitialDelay int64             `json:"initial_delay_ns"`
	MaxDelay     int64             `json:"max_delay_ns"`
}

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

	asynqTask, asynqOptions, err := toAsynqTask(task, options)
	if err != nil {
		return port.DispatchInfo{}, err
	}
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

func toAsynqTask(task port.QueueTask, options port.DispatchOptions) (*asynq.Task, []asynq.Option, error) {
	headers := make(map[string]string)
	if options.Retry.MaxRetries > 0 {
		policy, err := json.Marshal(retryPolicyWire{
			Backoff:      options.Retry.Backoff,
			InitialDelay: int64(options.Retry.InitialDelay),
			MaxDelay:     int64(options.Retry.MaxDelay),
		})
		if err != nil {
			return nil, nil, fmt.Errorf("encode retry policy: %w", err)
		}
		headers[retryPolicyHeader] = string(policy)
	}

	opts := []asynq.Option{
		asynq.Queue(options.Queue),
		asynq.Timeout(options.Timeout),
		asynq.MaxRetry(options.Retry.MaxRetries),
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
	return asynq.NewTaskWithHeaders(task.Type, task.Payload, headers), opts, nil
}

func dispatchInfo(info *asynq.TaskInfo) port.DispatchInfo {
	return port.DispatchInfo{
		ID:        info.ID,
		Queue:     info.Queue,
		Type:      info.Type,
		ProcessAt: info.NextProcessAt,
	}
}

// RetryDelay calculates the retry delay configured by the producer. Invalid or
// absent policy metadata falls back to Asynq's default exponential strategy.
func RetryDelay(retryCount int, _ error, task *asynq.Task) time.Duration {
	policy, err := parseRetryPolicy(task.Headers())
	if err != nil {
		return asynq.DefaultRetryDelayFunc(retryCount, nil, task)
	}
	return retryDelayFor(policy, retryCount)
}

func parseRetryPolicy(headers map[string]string) (retryPolicyWire, error) {
	raw, ok := headers[retryPolicyHeader]
	if !ok {
		return retryPolicyWire{}, fmt.Errorf("retry policy header is missing")
	}
	var policy retryPolicyWire
	if err := json.Unmarshal([]byte(raw), &policy); err != nil {
		return retryPolicyWire{}, fmt.Errorf("decode retry policy: %w", err)
	}
	if policy.InitialDelay <= 0 || policy.MaxDelay < policy.InitialDelay {
		return retryPolicyWire{}, fmt.Errorf("retry policy delays are invalid")
	}
	switch policy.Backoff {
	case port.RetryBackoffFixed, port.RetryBackoffLinear, port.RetryBackoffExponential:
		return policy, nil
	default:
		return retryPolicyWire{}, fmt.Errorf("retry policy backoff is invalid")
	}
}

func retryDelayFor(policy retryPolicyWire, retryCount int) time.Duration {
	initial := time.Duration(policy.InitialDelay)
	maximum := time.Duration(policy.MaxDelay)
	if retryCount < 1 {
		retryCount = 1
	}

	var delay time.Duration
	switch policy.Backoff {
	case port.RetryBackoffFixed:
		delay = initial
	case port.RetryBackoffLinear:
		delay = multiplyDuration(initial, retryCount)
	case port.RetryBackoffExponential:
		factor := math.Pow(2, float64(retryCount-1))
		if factor > float64(math.MaxInt64/int64(initial)) {
			delay = maximum
		} else {
			delay = time.Duration(float64(initial) * factor)
		}
	}
	if delay > maximum {
		return maximum
	}
	return delay
}

func multiplyDuration(value time.Duration, multiplier int) time.Duration {
	if multiplier > 0 && value > time.Duration(math.MaxInt64/int64(multiplier)) {
		return time.Duration(math.MaxInt64)
	}
	return value * time.Duration(multiplier)
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
				return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
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
		RetryDelayFunc:  RetryDelay,
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

func NewScheduler(redisClient *redis.Client, provider port.PeriodicTaskProvider, logger *slog.Logger) (*Scheduler, error) {
	if provider == nil {
		return nil, fmt.Errorf("periodic task provider is required")
	}
	redisOptions := redisClient.Options()
	s := asynq.NewScheduler(asynq.RedisClientOpt{
		Addr:     redisOptions.Addr,
		Username: redisOptions.Username,
		Password: redisOptions.Password,
		DB:       redisOptions.DB,
	}, &asynq.SchedulerOpts{
		Location: time.UTC,
		Logger:   queueLogger{logger: logger},
		LogLevel: asynq.WarnLevel,
	})
	for _, periodicTask := range provider.PeriodicTasks() {
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
		task, asynqOptions, err := toAsynqTask(periodicTask.Task, options)
		if err != nil {
			return nil, fmt.Errorf("create periodic task %q: %w", periodicTask.Name, err)
		}
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
