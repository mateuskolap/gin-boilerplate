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

	// Atomic Counter
	Incr(ctx context.Context, key string) (int64, error)

	// SetAddIfVersionMatch is used to prevent caching stale data when a concurrent invalidation
	// occurred between reading from DB and writing to cache.
	SetAddIfVersionMatch(ctx context.Context, setKey string, members []string, ttl time.Duration, versionKey string, expectedVersion int64) (bool, error)

	// Pattern Invalidation
	DeleteByPattern(ctx context.Context, pattern string) error
}
