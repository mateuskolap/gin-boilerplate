package repository

import (
	"context"
	"gin-boilerplate/internal/domain"
	"time"
	"uuid"

	"gorm.io/gorm"
)

type refreshTokenRepository struct {
	*baseRepository[domain.RefreshToken]
}

func NewRefreshTokenRepository(db *gorm.DB) domain.RefreshTokenRepository {
	return &refreshTokenRepository{
		baseRepository: newBaseRepository[domain.RefreshToken](db),
	}
}

func (r *refreshTokenRepository) FindByTokenHash(ctx context.Context, token string) (*domain.RefreshToken, error) {
	return r.FindOneBy(ctx, "token_hash = ?", []any{token})
}

func (r *refreshTokenRepository) ListByUserID(ctx context.Context, userID uuid.UUID, params domain.PaginationParams, filters []domain.Filter) (*domain.PaginatedResult[domain.RefreshToken], error) {
	sanitizedFilters := append(domain.Filters(filters).Without("user_id"), domain.Filter{
		Field:    "user_id",
		Operator: domain.OperatorEquals,
		Value:    userID,
	})

	return r.List(ctx, params, sanitizedFilters)
}

func (r *refreshTokenRepository) RevokeAllByUserID(ctx context.Context, userID uuid.UUID) error {
	return r.getDB(ctx).
		Model(&domain.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", time.Now().UTC()).
		Error
}

func (r *refreshTokenRepository) RevokeByID(ctx context.Context, id uuid.UUID, replacedBy uuid.UUID) (bool, error) {
	result := r.getDB(ctx).
		Model(&domain.RefreshToken{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Updates(map[string]any{
			"revoked_at":  time.Now().UTC(),
			"replaced_by": replacedBy,
		})

	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected > 0, nil
}
