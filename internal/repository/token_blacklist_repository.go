package repository

import (
	"context"
	"fmt"
	"gin-boilerplate/internal/domain"
	"time"
)

type tokenBlacklistRepository struct {
	cache domain.CacheRepository
}

func NewTokenBlackListRepository(cache domain.CacheRepository) domain.TokenBlackListRepository {
	return &tokenBlacklistRepository{
		cache: cache,
	}
}

func (b *tokenBlacklistRepository) RevokeToken(ctx context.Context, jti string, expiresIn time.Duration) error {
	key := fmt.Sprintf("blacklist:jti:%s", jti)

	return b.cache.Set(ctx, key, "revoked", expiresIn)
}

func (b *tokenBlacklistRepository) IsRevoked(ctx context.Context, jti string) (bool, error) {
	key := fmt.Sprintf("blacklist:jti:%s", jti)

	var val string
	if err := b.cache.Get(ctx, key, &val); err != nil {
		return false, err
	}

	return val != "", nil
}
