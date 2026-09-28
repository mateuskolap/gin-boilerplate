package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"gin-boilerplate/internal/domain/port"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

type testTaskHandler struct {
	taskType string
	payload  []byte
	err      error
}

func (h *testTaskHandler) TaskType() string { return h.taskType }

func (h *testTaskHandler) HandleTask(_ context.Context, payload json.RawMessage) error {
	h.payload = append([]byte(nil), payload...)
	return h.err
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestQueueLoggerForwardsAllLevels(t *testing.T) {
	var output bytes.Buffer
	logger := queueLogger{logger: slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug}))}
	logger.Debug("debug detail")
	logger.Info("info detail")
	logger.Warn("warn detail")
	logger.Error("error detail")
	logger.Fatal("fatal detail")
	for _, want := range []string{"level=DEBUG", "message=\"debug detail\"", "level=INFO", "message=\"info detail\"", "level=WARN", "message=\"warn detail\"", "level=ERROR", "message=\"error detail\"", "msg=\"asynq fatal\"", "message=\"fatal detail\""} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("queue logger output %q does not contain %q", output.String(), want)
		}
	}
}

func TestWorkerRegistersAndRunsTaskHandlers(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", MaxRetries: 0})
	t.Cleanup(func() { _ = client.Close() })
	handler := &testTaskHandler{taskType: "test.task"}
	worker, err := NewWorker(client, WorkerConfig{Concurrency: 1, ShutdownTimeout: time.Second}, testLogger(), handler)
	if err != nil {
		t.Fatalf("NewWorker() error = %v", err)
	}

	payload := []byte(`{"id":1}`)
	if err := worker.mux.ProcessTask(context.Background(), asynq.NewTask(handler.taskType, payload)); err != nil {
		t.Fatalf("ProcessTask() error = %v", err)
	}
	if string(handler.payload) != string(payload) {
		t.Fatalf("handler payload = %s, want %s", handler.payload, payload)
	}
	if err := worker.mux.ProcessTask(context.Background(), asynq.NewTask(handler.taskType, []byte("{"))); !errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("ProcessTask() invalid JSON error = %v, want SkipRetry", err)
	}
	if err := worker.mux.ProcessTask(context.Background(), asynq.NewTask("unknown.task", []byte(`{}`))); !errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("ProcessTask() unknown task error = %v, want SkipRetry", err)
	}
}

func TestWorkerRejectsInvalidAndDuplicateHandlers(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", MaxRetries: 0})
	t.Cleanup(func() { _ = client.Close() })
	config := WorkerConfig{Concurrency: 1, ShutdownTimeout: time.Second}
	valid := &testTaskHandler{taskType: "same.task"}
	for _, handlers := range [][]port.TaskHandler{
		{nil},
		{&testTaskHandler{}},
		{valid, valid},
	} {
		if _, err := NewWorker(client, config, testLogger(), handlers...); err == nil {
			t.Errorf("NewWorker() accepted invalid handlers: %#v", handlers)
		}
	}
}

func TestSchedulerValidatesAndRegistersPeriodicTasks(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", MaxRetries: 0})
	t.Cleanup(func() { _ = client.Close() })
	valid := port.PeriodicTask{
		Name:    "cleanup",
		Cron:    "0 3 * * *",
		Task:    port.QueueTask{Type: "test.cleanup", Payload: []byte(`{}`)},
		Options: port.DispatchOptions{Timeout: time.Minute, MaxRetries: 2},
	}
	scheduler, err := NewScheduler(client, []port.PeriodicTask{valid}, testLogger())
	if err != nil {
		t.Fatalf("NewScheduler() rejected valid task: %v", err)
	}
	if scheduler.scheduler == nil {
		t.Fatal("NewScheduler() returned an empty scheduler")
	}
	for _, invalid := range []port.PeriodicTask{
		{Name: "", Cron: "* * * * *", Task: valid.Task, Options: valid.Options},
		{Name: "bad cron", Cron: "not a cron", Task: valid.Task, Options: valid.Options},
		{Name: "bad task", Cron: "* * * * *", Task: port.QueueTask{Type: "", Payload: []byte(`{}`)}, Options: valid.Options},
	} {
		if _, err := NewScheduler(client, []port.PeriodicTask{invalid}, testLogger()); err == nil {
			t.Errorf("NewScheduler() accepted invalid task: %+v", invalid)
		}
	}
}

func TestQueueTaskConversionAndOptionDefaults(t *testing.T) {
	options := normalizeOptions(port.DispatchOptions{})
	if options.Queue != port.DefaultQueue {
		t.Fatalf("normalizeOptions() queue = %q", options.Queue)
	}
	processAt := time.Date(2026, 9, 27, 12, 0, 0, 0, time.FixedZone("UTC-3", -3*60*60))
	task, _ := toAsynqTask(port.QueueTask{Type: "test.task", Payload: []byte(`{"ok":true}`)}, port.DispatchOptions{Timeout: time.Minute, ProcessAt: &processAt})
	if task.Type() != "test.task" || string(task.Payload()) != `{"ok":true}` {
		t.Fatalf("toAsynqTask() type=%q payload=%s", task.Type(), task.Payload())
	}

	info := dispatchInfo(&asynq.TaskInfo{ID: "id", Queue: "maintenance", Type: "test.task", NextProcessAt: processAt.UTC()})
	if info.ID != "id" || info.Queue != "maintenance" || info.Type != "test.task" || !info.ProcessAt.Equal(processAt.UTC()) {
		t.Fatalf("dispatchInfo() = %+v", info)
	}
}

func TestDispatcherRejectsInvalidTaskAndOptionsBeforeEnqueue(t *testing.T) {
	dispatcher := &dispatcher{}
	if _, err := dispatcher.Dispatch(context.Background(), port.QueueTask{}, port.DispatchOptions{}); err == nil {
		t.Fatal("Dispatch() accepted an invalid task")
	}
	if _, err := dispatcher.Dispatch(context.Background(), port.QueueTask{Type: "test.task", Payload: []byte(`{}`)}, port.DispatchOptions{}); err == nil {
		t.Fatal("Dispatch() accepted options without a timeout")
	}
}
