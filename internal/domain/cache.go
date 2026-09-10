package domain

import (
	"context"
	"time"
)

type CacheRepository interface {
	// Set stores a value under the given key with an expiration duration.
	Set(ctx context.Context, key string, value any, ttl time.Duration) error

	// Get retrieves a value by key and unmarshals it into dest.
	// Returns nil if the key does not exist.
	Get(ctx context.Context, key string, dest any) error

	// Delete removes a key and its value from cache.
	Delete(ctx context.Context, key string) error
}
