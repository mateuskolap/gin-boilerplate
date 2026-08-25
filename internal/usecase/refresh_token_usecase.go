package usecase

import (
	"context"
	"gin-boilerplate/internal/domain"
	"gin-boilerplate/pkg/security"
	"time"
	"uuid"
)

type refreshTokenUseCase struct {
	refreshTokenRepo domain.RefreshTokenRepository
	expiration       time.Duration
}

func NewRefreshTokenUseCase(
	refreshTokenRepo domain.RefreshTokenRepository,
	expiration time.Duration,
) domain.RefreshTokenUseCase {
	return &refreshTokenUseCase{
		refreshTokenRepo: refreshTokenRepo,
		expiration:       expiration,
	}
}

func (r *refreshTokenUseCase) Create(ctx context.Context, token *domain.RefreshToken) (string, error) {
	plainToken, err := security.GenerateRandomToken(32)
	if err != nil {
		return "", domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to generate refresh token",
			err,
		)
	}

	token.TokenHash = security.HashSHA256(plainToken)

	if token.ExpiresAt.IsZero() {
		token.ExpiresAt = time.Now().UTC().Add(r.expiration)
	}

	if err := r.refreshTokenRepo.Create(ctx, token); err != nil {
		return "", domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to create refresh token",
			err,
		)
	}

	return plainToken, nil
}

func (r *refreshTokenUseCase) FindByTokenHash(ctx context.Context, token string) (*domain.RefreshToken, error) {
	panic("unimplemented")
}

func (r *refreshTokenUseCase) ListByUserID(ctx context.Context, userID uuid.UUID, params domain.PaginationParams, filters []domain.Filter) (*domain.PaginatedResult[domain.RefreshToken], error) {
	panic("unimplemented")
}

func (r *refreshTokenUseCase) Revoke(ctx context.Context, token string) error {
	panic("unimplemented")
}
