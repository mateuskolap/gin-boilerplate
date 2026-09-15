package usecase

import (
	"context"
	"errors"
	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/infra/security"
	"time"

	"uuid"
)

var allowedRefreshTokenFilterFields = map[string]bool{
	"user_agent": true,
	"expires_at": true,
	"ip_address": true,
	"created_at": true,
}

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

func (r *refreshTokenUseCase) Create(ctx context.Context, userID uuid.UUID, ipAddress string, userAgent string) (string, error) {
	expiresAt := time.Now().UTC().Add(r.expiration)
	plainToken, _, err := r.createTokenWithExpiry(ctx, userID, ipAddress, userAgent, expiresAt)
	return plainToken, err
}

func (r *refreshTokenUseCase) Rotate(ctx context.Context, oldToken string, ipAddress string, userAgent string) (string, error) {
	storedToken, err := r.Validate(ctx, oldToken)
	if err != nil {
		return "", err
	}

	newPlainToken, newTokenEntity, err := r.createTokenWithExpiry(ctx, storedToken.UserID, ipAddress, userAgent, storedToken.ExpiresAt)
	if err != nil {
		return "", err
	}

	revoked, err := r.refreshTokenRepo.RevokeByID(ctx, storedToken.ID, newTokenEntity.ID)
	if err != nil {
		return "", domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to revoke refresh token",
			err,
		)
	}

	if !revoked {
		_ = r.refreshTokenRepo.RevokeAllByUserID(ctx, storedToken.UserID)

		return "", domain.NewAppError(
			domain.ErrTypeUnauthorized,
			"Refresh token has already been used",
			nil,
		)
	}

	return newPlainToken, nil
}

func (r *refreshTokenUseCase) FindByTokenHash(ctx context.Context, token string) (*domain.RefreshToken, error) {
	tokenHash := security.HashSHA256(token)
	existingRefreshToken, err := r.refreshTokenRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, domain.NewAppError(
			domain.ErrTypeInternal,
			"There was a problem verifying the token existence",
			err,
		)
	}

	if existingRefreshToken == nil {
		return nil, domain.NewAppError(
			domain.ErrTypeNotFound,
			"Refresh token not found",
			nil,
		)
	}

	return existingRefreshToken, nil
}

func (r *refreshTokenUseCase) ListActiveByUserID(
	ctx context.Context,
	userID uuid.UUID,
	params domain.PaginationParams,
	filters []domain.Filter,
) (*domain.PaginatedResult[domain.RefreshToken], error) {
	if err := domain.Filters(filters).ValidateAllowed(allowedRefreshTokenFilterFields); err != nil {
		return nil, err
	}

	filters = append(append(domain.Filters(filters).Without("expires_at"), domain.Filter{
		Field:    "expires_at",
		Operator: domain.OperatorGreaterThan,
		Value:    time.Now().UTC(),
	}), domain.Filter{
		Field:    "revoked_at",
		Operator: domain.OperatorIsNull,
	})

	if err := params.ValidateSort(allowedRefreshTokenFilterFields); err != nil {
		return nil, err
	}

	result, err := r.refreshTokenRepo.ListByUserID(ctx, userID, params, filters)
	if err != nil {
		return nil, domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to list refresh tokens",
			err,
		)
	}

	return result, nil
}

func (r *refreshTokenUseCase) Revoke(ctx context.Context, token string) error {
	existingRefreshToken, err := r.FindByTokenHash(ctx, token)
	if err != nil {
		return err
	}

	return r.RevokeEntity(ctx, existingRefreshToken)
}

func (r *refreshTokenUseCase) RevokeEntity(ctx context.Context, refreshToken *domain.RefreshToken) error {
	if refreshToken.RevokedAt != nil {
		return nil
	}

	now := time.Now().UTC()
	refreshToken.RevokedAt = &now

	if err := r.refreshTokenRepo.Update(ctx, refreshToken); err != nil {
		return domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to revoke refresh token",
			err,
		)
	}

	return nil
}

func (r *refreshTokenUseCase) RevokeAllByUserID(ctx context.Context, userID uuid.UUID) error {
	if err := r.refreshTokenRepo.RevokeAllByUserID(ctx, userID); err != nil {
		return domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to revoke all refresh tokens for user",
			err,
		)
	}

	return nil
}

func (r *refreshTokenUseCase) Validate(ctx context.Context, token string) (*domain.RefreshToken, error) {
	if token == "" {
		return nil, domain.NewAppError(
			domain.ErrTypeValidation,
			"Refresh token is required",
			nil,
		)
	}

	storedToken, err := r.FindByTokenHash(ctx, token)
	if err != nil {
		var appErr *domain.AppError
		if errors.As(err, &appErr) && appErr.Type == domain.ErrTypeNotFound {
			return nil, domain.NewAppError(
				domain.ErrTypeUnauthorized,
				"Invalid refresh token",
				nil,
			)
		}

		return nil, err
	}

	if storedToken.RevokedAt != nil {
		_ = r.refreshTokenRepo.RevokeAllByUserID(ctx, storedToken.UserID)

		return nil, domain.NewAppError(
			domain.ErrTypeUnauthorized,
			"Refresh token has been revoked",
			nil,
		)
	}

	if time.Now().UTC().After(storedToken.ExpiresAt) {
		return nil, domain.NewAppError(
			domain.ErrTypeUnauthorized,
			"Refresh token has expired",
			nil,
		)
	}

	return storedToken, nil
}

func (r *refreshTokenUseCase) createTokenWithExpiry(
	ctx context.Context,
	userID uuid.UUID,
	ipAddress,
	userAgent string,
	expiresAt time.Time,
) (string, *domain.RefreshToken, error) {
	plainToken, err := security.GenerateRandomToken(32)
	if err != nil {
		return "", nil, domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to generate refresh token",
			err,
		)
	}

	token := &domain.RefreshToken{
		UserID:    userID,
		TokenHash: security.HashSHA256(plainToken),
		ExpiresAt: expiresAt,
		IpAddress: ipAddress,
		UserAgent: userAgent,
	}

	if err := r.refreshTokenRepo.Create(ctx, token); err != nil {
		return "", nil, domain.NewAppError(
			domain.ErrTypeInternal,
			"Failed to create refresh token",
			err,
		)
	}

	return plainToken, token, nil
}
