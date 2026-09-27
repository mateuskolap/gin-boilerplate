package port

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const DefaultQueue = "default"

// DispatchOptions configures the lifecycle of a queued task.
type DispatchOptions struct {
	Queue     string
	ProcessAt *time.Time
	Timeout   time.Duration
	// MaxRetries is the number of attempts after the first failed execution.
	MaxRetries int
	UniqueFor  time.Duration
	Retention  time.Duration
}

// QueueTask is the transport-neutral representation of a background task.
type QueueTask struct {
	Type    string
	Payload json.RawMessage
}

type DispatchInfo struct {
	ID        string
	Queue     string
	Type      string
	ProcessAt time.Time
}

// QueueDispatcher sends tasks to the configured asynchronous queue.
type QueueDispatcher interface {
	Dispatch(ctx context.Context, task QueueTask, options DispatchOptions) (DispatchInfo, error)
}

// TaskHandler is implemented by application handlers and invoked by a worker.
type TaskHandler interface {
	TaskType() string
	HandleTask(ctx context.Context, payload json.RawMessage) error
}

// PeriodicTask is a cron declaration owned by the application.
type PeriodicTask struct {
	Name    string
	Cron    string
	Task    QueueTask
	Options DispatchOptions
}

// PeriodicTaskProvider supplies the static recurring tasks to the scheduler.
type PeriodicTaskProvider interface {
	PeriodicTasks() []PeriodicTask
}

func (t QueueTask) Validate() error {
	if strings.TrimSpace(t.Type) == "" {
		return fmt.Errorf("queue task type must not be empty")
	}
	if len(t.Payload) == 0 || !json.Valid(t.Payload) {
		return fmt.Errorf("queue task payload must be valid JSON")
	}
	return nil
}

func (o DispatchOptions) Validate() error {
	if o.Queue == "" {
		o.Queue = DefaultQueue
	}
	if strings.TrimSpace(o.Queue) == "" {
		return fmt.Errorf("queue name must not be empty")
	}
	if o.Timeout <= 0 {
		return fmt.Errorf("queue task timeout must be greater than zero")
	}
	if o.UniqueFor < 0 || o.Retention < 0 {
		return fmt.Errorf("queue task durations must not be negative")
	}
	if o.MaxRetries < 0 {
		return fmt.Errorf("queue max retries must not be negative")
	}
	return nil
}
