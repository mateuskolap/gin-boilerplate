package jobs

import (
	"context"
	"encoding/json"
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

func (h *purgeExpiredRefreshTokensHandler) HandleTask(ctx context.Context, _ json.RawMessage) error {
	_, err := h.refreshTokens.PurgeExpiredBefore(ctx, time.Now().UTC().Add(-h.retention))
	return err
}

func MaintenanceTasks() []port.PeriodicTask {
	return []port.PeriodicTask{{
		Name: "purge-expired-refresh-tokens",
		Cron: "0 3 * * *",
		Task: port.QueueTask{Type: PurgeExpiredRefreshTokensTask, Payload: json.RawMessage(`{}`)},
		Options: port.DispatchOptions{
			Queue:      "maintenance",
			Timeout:    time.Minute,
			MaxRetries: 3,
			UniqueFor:  24 * time.Hour,
		},
	}}
}
