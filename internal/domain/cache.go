package domain

import (
	"context"
	"time"
)

type CacheRepository interface {
	// Standard Key-Value
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Get(ctx context.Context, key string, dest any) error
	Delete(ctx context.Context, key string) error

	// Native Sets
	SetAdd(ctx context.Context, key string, members []string, ttl time.Duration) error
	SetMembers(ctx context.Context, key string) ([]string, error)
	CheckSetMembers(ctx context.Context, keys []string, member string) (hasMember bool, missingKeys []string, err error)

	// Pattern Invalidation
	DeleteByPattern(ctx context.Context, pattern string) error
}
