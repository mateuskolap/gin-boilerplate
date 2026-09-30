package postgres

import (
	"context"
	"gin-boilerplate/internal/domain/shared"
	postgresinfra "gin-boilerplate/internal/infra/postgres"
	refreshtokendomain "gin-boilerplate/internal/refresh_tokens/domain"
	"time"
	"uuid"

	"gorm.io/gorm"
)

type refreshTokenRepository struct {
	*postgresinfra.BaseRepository[refreshtokendomain.RefreshToken]
}

func NewRefreshTokenRepository(db *gorm.DB) refreshtokendomain.RefreshTokenRepository {
	return &refreshTokenRepository{
		BaseRepository: postgresinfra.NewBaseRepository(db,
			func() any { return &RefreshTokenModel{} },
			func(token *refreshtokendomain.RefreshToken) any { return refreshTokenModelFromDomain(token) },
			refreshTokenDomainFromModel,
		),
	}
}

func (r *refreshTokenRepository) FindByTokenHash(ctx context.Context, token string) (*refreshtokendomain.RefreshToken, error) {
	return r.FindOneBy(ctx, "token_hash = ?", []any{token})
}

func (r *refreshTokenRepository) ListByUserID(ctx context.Context, userID uuid.UUID, params shared.PaginationParams, filters []shared.Filter) (*shared.PaginatedResult[refreshtokendomain.RefreshToken], error) {
	sanitizedFilters := append(shared.Filters(filters).Without("user_id"), shared.Filter{
		Field:    "user_id",
		Operator: shared.OperatorEquals,
		Value:    userID,
	})

	return r.List(ctx, params, sanitizedFilters)
}

func (r *refreshTokenRepository) RevokeAllByUserID(ctx context.Context, userID uuid.UUID) error {
	return r.DB(ctx).
		Model(&RefreshTokenModel{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", time.Now().UTC()).
		Error
}

func (r *refreshTokenRepository) Revoke(ctx context.Context, id, userID uuid.UUID, replacedBy *uuid.UUID) (bool, error) {
	now := time.Now().UTC()
	updates := map[string]any{"revoked_at": now}
	if replacedBy != nil {
		updates["replaced_by"] = *replacedBy
	}

	result := r.DB(ctx).
		Model(&RefreshTokenModel{}).
		Where("id = ? AND user_id = ? AND revoked_at IS NULL AND expires_at > ?", id, userID, now).
		Updates(updates)

	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected > 0, nil
}

func (r *refreshTokenRepository) RevokeAllExcept(ctx context.Context, userID, exceptID uuid.UUID) error {
	return r.DB(ctx).
		Model(&RefreshTokenModel{}).
		Where("user_id = ? AND id <> ? AND revoked_at IS NULL AND expires_at > ?", userID, exceptID, time.Now().UTC()).
		Update("revoked_at", time.Now().UTC()).
		Error
}

func (r *refreshTokenRepository) DeleteExpiredBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	result := r.DB(ctx).
		Where("expires_at < ?", cutoff).
		Delete(&RefreshTokenModel{})
	return result.RowsAffected, result.Error
}
