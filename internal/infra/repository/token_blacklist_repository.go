package repository

import (
	"context"
	"fmt"
	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/domain/port"
	"time"
)

type tokenBlacklistRepository struct {
	cache port.CacheRepository
}

func NewTokenBlackListRepository(cache port.CacheRepository) domain.TokenBlackListRepository {
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

func (b *tokenBlacklistRepository) RevokeUserTokens(ctx context.Context, userID string, expiresIn time.Duration) error {
	key := fmt.Sprintf("blacklist:user:%s", userID)
	revokedAt := time.Now().UTC().Unix()
	return b.cache.Set(ctx, key, revokedAt, expiresIn)
}

func (b *tokenBlacklistRepository) IsUserTokenRevoked(ctx context.Context, userID string, issuedAt time.Time) (bool, error) {
	key := fmt.Sprintf("blacklist:user:%s", userID)

	var revokedAt int64
	if err := b.cache.Get(ctx, key, &revokedAt); err != nil {
		return false, err
	}

	if revokedAt == 0 {
		return false, nil
	}

	return issuedAt.Unix() <= revokedAt, nil
}
