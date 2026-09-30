package application

import (
	"context"
	"fmt"
	"time"

	refreshtokendomain "gin-boilerplate/internal/refresh_tokens/domain"
)

type refreshTokenMaintenanceUseCase struct {
	refreshTokenRepo refreshtokendomain.RefreshTokenRepository
}

func NewRefreshTokenMaintenanceUseCase(refreshTokenRepo refreshtokendomain.RefreshTokenRepository) refreshtokendomain.RefreshTokenMaintenanceUseCase {
	return &refreshTokenMaintenanceUseCase{refreshTokenRepo: refreshTokenRepo}
}

func (u *refreshTokenMaintenanceUseCase) PurgeExpiredBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	if cutoff.IsZero() {
		return 0, fmt.Errorf("refresh token purge cutoff must be set")
	}
	return u.refreshTokenRepo.DeleteExpiredBefore(ctx, cutoff.UTC())
}
