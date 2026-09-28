package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"gin-boilerplate/internal/domain/port"
)

type purgeUseCase struct {
	err    error
	cutoff time.Time
	calls  int
}

func (f *purgeUseCase) PurgeExpiredBefore(_ context.Context, cutoff time.Time) (int64, error) {
	f.calls++
	f.cutoff = cutoff
	return 5, f.err
}

func TestPurgeExpiredRefreshTokensHandlerUsesRetentionCutoff(t *testing.T) {
	retention := 48 * time.Hour
	fake := &purgeUseCase{}
	handler := NewPurgeExpiredRefreshTokensHandler(fake, retention)
	started := time.Now().UTC()
	if got := handler.TaskType(); got != PurgeExpiredRefreshTokensTask {
		t.Fatalf("TaskType() = %q", got)
	}
	if err := handler.HandleTask(context.Background(), json.RawMessage(`{}`)); err != nil {
		t.Fatalf("HandleTask() error = %v", err)
	}
	if fake.calls != 1 || fake.cutoff.Before(started.Add(-retention)) || fake.cutoff.After(time.Now().UTC().Add(-retention)) {
		t.Fatalf("HandleTask() calls=%d cutoff=%v", fake.calls, fake.cutoff)
	}
}

func TestMaintenanceTasksDeclareValidDailySchedule(t *testing.T) {
	tasks := MaintenanceTasks()
	if len(tasks) != 1 {
		t.Fatalf("MaintenanceTasks() returned %d tasks", len(tasks))
	}
	task := tasks[0]
	if task.Name != "purge-expired-refresh-tokens" || task.Cron != "0 3 * * *" || task.Task.Type != PurgeExpiredRefreshTokensTask || task.Options.Queue != "maintenance" || task.Options.Timeout != time.Minute || task.Options.MaxRetries != 3 || task.Options.UniqueFor != 24*time.Hour {
		t.Fatalf("unexpected maintenance schedule: %+v", task)
	}
	if err := task.Task.Validate(); err != nil {
		t.Fatalf("scheduled payload is invalid: %v", err)
	}
	if err := task.Options.Validate(); err != nil {
		t.Fatalf("scheduled options are invalid: %v", err)
	}
}

func TestPurgeHandlerReturnsMaintenanceError(t *testing.T) {
	want := errors.New("database unavailable")
	handler := NewPurgeExpiredRefreshTokensHandler(&purgeUseCase{err: want}, time.Hour)
	if err := handler.HandleTask(context.Background(), nil); !errors.Is(err, want) {
		t.Fatalf("HandleTask() error = %v, want %v", err, want)
	}
}

var _ port.TaskHandler = (*purgeExpiredRefreshTokensHandler)(nil)
