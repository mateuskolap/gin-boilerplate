package usecase

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRefreshTokenMaintenancePassesUTCcutoffToRepository(t *testing.T) {
	repo := &testRefreshRepo{deleteCount: 3}
	useCase := NewRefreshTokenMaintenanceUseCase(repo)
	cutoff := time.Date(2026, time.September, 1, 12, 0, 0, 0, time.FixedZone("UTC-3", -3*60*60))

	deleted, err := useCase.PurgeExpiredBefore(context.Background(), cutoff)
	if err != nil || deleted != 3 || !repo.deleteBefore.Equal(cutoff.UTC()) {
		t.Fatalf("PurgeExpiredBefore() deleted=%d cutoff=%v error=%v", deleted, repo.deleteBefore, err)
	}
}

func TestRefreshTokenMaintenanceRejectsUnsetCutoffAndPropagatesErrors(t *testing.T) {
	repo := &testRefreshRepo{deleteErr: errors.New("database unavailable")}
	useCase := NewRefreshTokenMaintenanceUseCase(repo)
	if _, err := useCase.PurgeExpiredBefore(context.Background(), time.Time{}); err == nil {
		t.Fatal("PurgeExpiredBefore() accepted a zero cutoff")
	}
	if _, err := useCase.PurgeExpiredBefore(context.Background(), time.Now()); err == nil || repo.deleteBefore.IsZero() {
		t.Fatalf("PurgeExpiredBefore() error=%v cutoff=%v", err, repo.deleteBefore)
	}
}
