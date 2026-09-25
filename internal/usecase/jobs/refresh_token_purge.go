package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/domain/port"
)

const PurgeExpiredRefreshTokensTask = "maintenance.refresh_tokens.purge"

type purgeExpiredRefreshTokensHandler struct {
	refreshTokens domain.RefreshTokenMaintenanceUseCase
	retention     time.Duration
}

func NewPurgeExpiredRefreshTokensHandler(
	refreshTokens domain.RefreshTokenMaintenanceUseCase,
	retention time.Duration,
) port.TaskHandler {
	return &purgeExpiredRefreshTokensHandler{
		refreshTokens: refreshTokens,
		retention:     retention,
	}
}

func (h *purgeExpiredRefreshTokensHandler) TaskType() string {
	return PurgeExpiredRefreshTokensTask
}

func (h *purgeExpiredRefreshTokensHandler) HandleTask(ctx context.Context, payload json.RawMessage) error {
	var request struct{}
	if err := json.Unmarshal(payload, &request); err != nil {
		return fmt.Errorf("decode refresh token purge task: %w", err)
	}
	_, err := h.refreshTokens.PurgeExpiredBefore(ctx, time.Now().UTC().Add(-h.retention))
	return err
}

type maintenanceSchedule struct{}

func NewMaintenanceSchedule() port.PeriodicTaskProvider {
	return maintenanceSchedule{}
}

func (maintenanceSchedule) PeriodicTasks() []port.PeriodicTask {
	return []port.PeriodicTask{{
		Name: "purge-expired-refresh-tokens",
		Cron: "0 3 * * *",
		Task: port.QueueTask{Type: PurgeExpiredRefreshTokensTask, Payload: json.RawMessage(`{}`)},
		Options: port.DispatchOptions{
			Queue:   "maintenance",
			Timeout: time.Minute,
			Retry: port.RetryPolicy{
				MaxRetries:   3,
				Backoff:      port.RetryBackoffExponential,
				InitialDelay: time.Minute,
				MaxDelay:     time.Hour,
			},
			UniqueFor: 24 * time.Hour,
		},
	}}
}
