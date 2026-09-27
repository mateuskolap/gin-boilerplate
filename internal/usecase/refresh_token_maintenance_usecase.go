package usecase

import (
	"context"
	"fmt"
	"time"

	"gin-boilerplate/internal/domain"
)

type refreshTokenMaintenanceUseCase struct {
	refreshTokenRepo domain.RefreshTokenRepository
}

func NewRefreshTokenMaintenanceUseCase(refreshTokenRepo domain.RefreshTokenRepository) domain.RefreshTokenMaintenanceUseCase {
	return &refreshTokenMaintenanceUseCase{refreshTokenRepo: refreshTokenRepo}
}

func (u *refreshTokenMaintenanceUseCase) PurgeExpiredBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	if cutoff.IsZero() {
		return 0, fmt.Errorf("refresh token purge cutoff must be set")
	}
	return u.refreshTokenRepo.DeleteExpiredBefore(ctx, cutoff.UTC())
}
