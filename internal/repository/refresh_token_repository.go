package repository

import (
	"context"
	"gin-boilerplate/internal/domain"
	"uuid"

	"gorm.io/gorm"
)

type refreshTokenRepository struct {
	domain.BaseRepository[domain.RefreshToken]
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) domain.RefreshTokenRepository {
	return &refreshTokenRepository{
		BaseRepository: NewBaseRepository[domain.RefreshToken](db),
		db:             db,
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
